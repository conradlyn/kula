package collector

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	// PostgreSQL driver — imported for side-effect registration.
	_ "github.com/lib/pq"
)

// pgRaw holds the raw cumulative counters from pg_stat_database.
type pgRaw struct {
	xactCommit   int64
	xactRollback int64
	tupFetched   int64
	tupInserted  int64
	tupUpdated   int64
	tupDeleted   int64
	blksRead     int64
	blksHit      int64
}

// postgresCollector manages the PostgreSQL connection and metrics.
type postgresCollector struct {
	dsn    string
	db     *sql.DB
	dbName string
	prev   pgRaw
	debug  bool
}

// newPostgresCollector builds the DSN and returns a collector (without connecting yet).
// Connection is lazy — established on first Collect() call.
func newPostgresCollector(host string, port int, user, password, dbname, sslmode string, debug bool) *postgresCollector {
	var dsn string
	if port == 0 {
		// Unix socket mode: host is the socket directory
		dsn = fmt.Sprintf("host=%s user=%s dbname=%s sslmode=%s",
			host, user, dbname, sslmode)
	} else {
		dsn = fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s",
			host, port, user, dbname, sslmode)
	}
	if password != "" {
		dsn += fmt.Sprintf(" password=%s", password)
	}

	return &postgresCollector{
		dsn:    dsn,
		dbName: dbname,
		debug:  debug,
	}
}

// connect establishes (or verifies) the DB connection.
func (pc *postgresCollector) connect() error {
	if pc.db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := pc.db.PingContext(ctx); err == nil {
			return nil // already connected
		}
		// Connection lost, close and retry
		_ = pc.db.Close()
		pc.db = nil
	}

	db, err := sql.Open("postgres", pc.dsn)
	if err != nil {
		return fmt.Errorf("postgres open: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("postgres ping: %w", err)
	}
	pc.db = db
	if pc.debug {
		log.Printf("[postgres] connected to database %q (DSN length: %d)", pc.dbName, len(pc.dsn))
	}
	return nil
}

// Close closes the database connection.
func (pc *postgresCollector) Close() {
	if pc.db != nil {
		_ = pc.db.Close()
	}
}

// collectPostgres gathers PostgreSQL metrics. Returns nil on any error.
func (c *Collector) collectPostgres(elapsed float64) *PostgresStats {
	if c.pgCollector == nil {
		return nil
	}

	if err := c.pgCollector.connect(); err != nil {
		c.appErrorf("[postgres] connection failed: %v", err)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stats := &PostgresStats{}

	// Active and idle connections from pg_stat_activity
	c.debugf("[postgres] querying pg_stat_activity and pg_stat_database for %q", c.pgCollector.dbName)
	row := c.pgCollector.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN state = 'active' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN state = 'idle' THEN 1 ELSE 0 END), 0)
		FROM pg_stat_activity
		WHERE backend_type = 'client backend'
	`)
	if err := row.Scan(&stats.ActiveConns, &stats.IdleConns); err != nil {
		c.debugf("[postgres] scan activity error: %v", err)
		return nil
	}

	// Max connections
	var maxConnsStr string
	if err := c.pgCollector.db.QueryRowContext(ctx, "SHOW max_connections").Scan(&maxConnsStr); err == nil {
		if v, err := strconv.Atoi(maxConnsStr); err == nil {
			stats.MaxConns = v
		}
	}

	// Database-level cumulative counters from pg_stat_database
	var cur pgRaw
	row = c.pgCollector.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(xact_commit, 0),
			COALESCE(xact_rollback, 0),
			COALESCE(tup_fetched, 0),
			COALESCE(tup_inserted, 0),
			COALESCE(tup_updated, 0),
			COALESCE(tup_deleted, 0),
			COALESCE(blks_read, 0),
			COALESCE(blks_hit, 0)
		FROM pg_stat_database
		WHERE datname = $1
	`, c.pgCollector.dbName)
	if err := row.Scan(
		&cur.xactCommit, &cur.xactRollback,
		&cur.tupFetched, &cur.tupInserted, &cur.tupUpdated, &cur.tupDeleted,
		&cur.blksRead, &cur.blksHit,
	); err != nil {
		c.appErrorf("[postgres] query error: %v", err)
		return nil
	}

	c.debugf("[postgres] raw: commit=%d, rollback=%d, hit=%d, read=%d",
		cur.xactCommit, cur.xactRollback, cur.blksHit, cur.blksRead)

	c.pgCollector.calculateStats(stats, cur, elapsed)

	// Dead tuples across all user tables
	var deadTuples sql.NullInt64
	if err := c.pgCollector.db.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(n_dead_tup), 0) FROM pg_stat_user_tables",
	).Scan(&deadTuples); err == nil && deadTuples.Valid {
		stats.DeadTuples = deadTuples.Int64
	}

	// Database size
	var dbSize sql.NullInt64
	if err := c.pgCollector.db.QueryRowContext(ctx,
		"SELECT pg_database_size($1)", c.pgCollector.dbName,
	).Scan(&dbSize); err == nil && dbSize.Valid {
		stats.DBSizeBytes = dbSize.Int64
	}

	return stats
}

// calculateStats computes rates and ratios from raw counters.
func (pc *postgresCollector) calculateStats(stats *PostgresStats, cur pgRaw, elapsed float64) {
	// Compute per-second rates
	if pc.prev.xactCommit > 0 && elapsed > 0 {
		stats.TxCommitPS = round2(float64(cur.xactCommit-pc.prev.xactCommit) / elapsed)
		stats.TxRollbackPS = round2(float64(cur.xactRollback-pc.prev.xactRollback) / elapsed)
		stats.TupFetchedPS = round2(float64(cur.tupFetched-pc.prev.tupFetched) / elapsed)
		stats.TupInsertedPS = round2(float64(cur.tupInserted-pc.prev.tupInserted) / elapsed)
		stats.TupUpdatedPS = round2(float64(cur.tupUpdated-pc.prev.tupUpdated) / elapsed)
		stats.TupDeletedPS = round2(float64(cur.tupDeleted-pc.prev.tupDeleted) / elapsed)
	}
	pc.prev = cur

	// Buffer hit ratio
	totalBlks := cur.blksRead + cur.blksHit
	if totalBlks > 0 {
		stats.BlksHitPct = round2(float64(cur.blksHit) / float64(totalBlks) * 100)
	}
}
