# Embedded Storage Options for Kula

The user has inquired about switching away from our custom JSON Ring Buffer to a fully-fledged embedded database engine. When evaluating storage engines for highly resource-constrained environments (1-2 vCPU instances) that also need to be self-contained and easily cross-compiled (No CGO), here is the breakdown of our options:

## Option 1: SQLite (`modernc.org/sqlite`)
SQLite is the industry standard for embedded databases.

- **Pros:** Full SQL support, powerful aggregation, very fast queries by index (timestamp). A pure-Go port exists (`modernc.org/sqlite`), so cross-compilation on ARM64 and RISC-V works without C C-compilers.
- **Cons:** **Vacuum Spikes**. SQLite is not a time-series database. Because Kula needs a rolling time window (e.g., throwing away data older than 3 days), we would constantly need to `DELETE FROM metrics WHERE ts < X`. This leaves fragmented space on disk, which doesn't shrink the file until we manually run `VACUUM`. Running `VACUUM` locks the database and causes massive I/O and CPU spikes, especially on slow virtual servers. 

## Option 2: Key-Value Stores (`bbolt` or `badger`)
These are mature, pure-Go database engines. `bbolt` uses B+ trees (good for reads), and `Badger` uses LSM trees (excellent for massive write loads).

- **Pros:** Pure Go, highly optimized, no CGO compilation issues.
- **Cons:** **Garbage Collection and Compaction**. Badger relies heavily on background garbage collection which continuously compacts data. On a 1 vCPU machine, this will constantly eat away at available processing power. `bbolt` requires manual deletion of old keys and its own form of manual database compaction, similar to SQLite's `VACUUM`. They do not "wrap around" natively.

## Option 3: Custom Binary TSDB Ring Buffer
The core architecture of Kula's current [tier.go](file:///home/c0m4r/ai/kula/internal/storage/tier.go) (a Ring Buffer that simply wraps around and overwrites the oldest data when the file reaches the maximum size) is actually **the most elegant and efficient strategy** for a constrained environment. It guarantees zero CPU/I/O spikes from garbage collection, compaction, or vacuuming. It never exceeds its maximum size.

The only reason it is currently slow is that it uses variable-length **JSON**.

If we want database-level speeds without database-level overhead, we can write a **Fixed-Size Binary Format**:
- Each metric sample encodes to the exact same number of bytes using standard `encoding/binary`.
- Because the record length is perfectly predictable, we can use **Binary Search** ([O(log N)](file:///home/c0m4r/ai/kula/internal/storage/tier.go#45-89)) to find the requested timestamp instantly, instead of linearly scanning ([O(N)](file:///home/c0m4r/ai/kula/internal/storage/tier.go#45-89)) gigabytes of data.
- **Pros:** Zero dependencies, absolutely minimal CPU usage, parses data using zero-allocation arrays, flat CPU footprint over time.
- **Cons:** Requires us to hand-write a basic fixed-size binary codec instead of simply calling `json.Marshal()`.

## Next Steps
How would you like to proceed?
1. **Adopt an Embedded DB (e.g., pure-Go SQLite or bbolt):** Offload storage logic to a tested engine, accepting that background compaction/vacuuming will occasionally cause CPU or I/O spikes.
2. **Upgrade the current Ring Buffer to a Binary TSDB:** Maintain the zero-maintenance, zero-compaction architecture, but rewrite it to use fixed-size binary format and binary search for database-level query speeds. 
3. **Stick to the JSON hotfix:** Easiest to implement immediately, heavily optimizes the current approach by bypassing `json.Unmarshal`, but still linearly scans data.

Please let me know which path aligns best with Kula's long-term vision.
