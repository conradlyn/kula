# 🔍 Code Review: Kula (Linux Server Monitoring Tool)

Based on my analysis of the repository, here's a comprehensive code review covering **Code Quality**, **Performance**, and **Security**.

---

## 📊 Project Overview

**Kula** is a lightweight, self-contained Linux server monitoring tool written in Go that:
- Collects system metrics from `/proc` and `/sys`
- Uses a tiered ring-buffer storage engine
- Serves data via Web UI (WebSocket + REST API) and TUI
- Has optional authentication with Argon2id password hashing

---

## ✅ CODE QUALITY

### Strengths

| Area | Observation |
|------|-------------|
| **Code Organization** | Clean separation of concerns with `internal/` packages for collector, storage, web, config, sandbox |
| **Documentation** | Excellent README with architecture diagrams, installation guides, and configuration examples |
| **Error Handling** | Consistent error wrapping with `fmt.Errorf("...: %w", err)` pattern |
| **Testing Infrastructure** | Has `check.sh` script with `go vet`, `go test -race`, `govulncheck`, and `golangci-lint` |
| **Configuration** | Well-structured YAML config with sensible defaults and validation |

### Issues & Recommendations

#### 1. **Inconsistent Error Handling** ⚠️

```go
// In web/server.go - Silent error ignored
if err := gz.Close(); err != nil {
    // Error silently ignored
}

// In collector/cpu.go - Error ignored
_ = f.Close()
```

**Recommendation:** Log ignored errors or use `defer` with named return values for cleanup.

#### 2. **Magic Numbers** ⚠️

```go
// In storage/tier.go
const headerSize = 64  // OK - documented
// But elsewhere:
if len(fields) < 14 { continue }  // Why 14?
conn.SetReadLimit(4096)  // Why 4096?
```

**Recommendation:** Define named constants with explanations.

#### 3. **Race Condition Risk** ⚠️

```go
// In web/server.go - WebSocket hub
func (h *wsHub) broadcast(data []byte) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    for client := range h.clients {
        if !client.paused {
            select {
            case client.sendCh <- data:  // Non-blocking, may drop data
            default:
                // Client too slow, skip
            }
        }
    }
}
```

**Issue:** The `client.paused` check happens outside the client's mutex, creating a potential race condition.

**Recommendation:**
```go
client.mu.Lock()
paused := client.paused
client.mu.Unlock()
if !paused { ... }
```

#### 4. **Missing Input Validation** ⚠️

```go
// In web/server.go - History API
pointsStr := r.URL.Query().Get("points")
points := 450
if pointsStr != "" {
    _, _ = fmt.Sscanf(pointsStr, "%d", &points)  // No validation!
}
```

**Recommendation:** Validate range (e.g., 1-7200) and return error on invalid input.

#### 5. **Code Duplication** ⚠️

The `minSample()` and `maxSample()` functions in `storage/store.go` have significant duplication (~200 lines each).

**Recommendation:** Consider using generics (Go 1.18+) or a helper function to reduce duplication.

---

## ⚡ PERFORMANCE

### Strengths

| Area | Observation |
|------|-------------|
| **Ring Buffer Storage** | Pre-allocated files with O(1) writes, predictable memory usage |
| **Latest Sample Cache** | `latestCache` provides O(1) access to most recent data |
| **Buffered Reads** | Uses `bufio.NewReaderSize` with 1MB buffer for tier file reads |
| **Zero Dependencies** | Single binary deployment reduces overhead |
| **Compression** | Optional gzip compression for HTTP/WebSocket traffic |

### Issues & Recommendations

#### 1. **Header Write Frequency** ⚠️

```go
// In storage/tier.go
if t.count%10 == 0 {
    return t.writeHeader()  // Every 10 writes
}
```

**Issue:** On a busy system with 1-second collection, this writes the header every 10 seconds. Could be optimized.

**Recommendation:** Consider writing header on `Close()` and `Flush()` only, or use async writes.

#### 2. **Memory Allocation in Hot Path** ⚠️

```go
// In collector/cpu.go - Every collection cycle
func parseProcStat() []cpuRaw {
    var result []cpuRaw  // Unbounded slice growth
    // ...
}
```

**Recommendation:** Pre-allocate with expected capacity:
```go
result := make([]cpuRaw, 0, runtime.NumCPU()+1)
```

#### 3. **WebSocket Write Deadline** ⚠️

```go
// In web/websocket.go
_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
```

**Issue:** 10-second deadline may be too aggressive for slow clients, causing unnecessary disconnections.

**Recommendation:** Make configurable or increase to 30 seconds.

#### 4. **No Connection Pooling** ℹ️

Each HTTP request creates new file handles for `/proc` and `/sys` reads.

**Recommendation:** For high-frequency access, consider keeping file descriptors open (though `/proc` files are special).

#### 5. **Aggregation Buffer Reconstruction** ⚠️

```go
// In storage/store.go - On every startup
s.reconstructAggregationState()
```

**Issue:** Reads tail of lower tiers on startup, which could be slow for large tier files.

**Recommendation:** Store aggregation state in header to avoid reconstruction.

---

## 🔒 SECURITY

### Strengths

| Area | Observation |
|------|-------------|
| **Landlock Sandbox** | Uses Linux Landlock LSM to restrict filesystem and network access |
| **Argon2id Password Hashing** | Industry-standard password hashing with configurable parameters |
| **Constant-Time Comparison** | Uses `crypto/subtle.ConstantTimeCompare` for password validation |
| **CSP Headers** | Content-Security-Policy with nonce for script execution |
| **Secure Cookies** | HttpOnly, Secure (when HTTPS), SameSite=Strict |
| **Rate Limiting** | Login attempts limited to 5 per 5 minutes per IP |
| **Session Validation** | IP and User-Agent fingerprinting for sessions |
| **WebSocket Origin Check** | Prevents Cross-Site WebSocket Hijacking (CSWSH) |

### Issues & Recommendations

#### 1. **Critical: TrustProxy Configuration Risk** 🔴

```go
// In config.example.yaml
trust_proxy: false  # Default is FALSE - good!

// In web/server.go
func getClientIP(r *http.Request, trustProxy bool) string {
    if trustProxy {
        if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
            return strings.TrimSpace(strings.SplitN(xff, ",", 2)[0])
        }
    }
    // ...
}
```

**Issue:** When `trust_proxy: true`, the X-Forwarded-For header is trusted without validation. An attacker could spoof their IP.

**Recommendation:** 
- Add configuration for trusted proxy IPs
- Validate that requests come from trusted proxy addresses before trusting headers

#### 2. **Session Storage Permissions** ⚠️

```go
// In web/auth.go
path := filepath.Join(a.storageDir, "sessions.json")
return os.WriteFile(path, data, 0600)  // Good - 0600 permissions
```

**Issue:** The storage directory itself is created with `0755` permissions:
```go
os.MkdirAll(absDir, 0755)  // Should be 0700 for sensitive data
```

**Recommendation:** Change storage directory permissions to `0700`.

#### 3. **Password Hash Parameters** ⚠️

```go
// In config.example.yaml
argon2:
  time: 1      # OWASP recommends 2-4 for 2023+
  memory: 65536  # 64MB - acceptable
  threads: 4
```

**Issue:** Time cost of 1 is below OWASP recommendations for new applications.

**Recommendation:** Increase default to `time: 3` for better brute-force resistance.

#### 4. **Missing Security Headers** ⚠️

```go
// In web/server.go
w.Header().Set("X-Content-Type-Options", "nosniff")
w.Header().Set("X-Frame-Options", "DENY")
w.Header().Set("Content-Security-Policy", ...)
// Missing:
// - Strict-Transport-Security (HSTS)
// - X-XSS-Protection (deprecated but still useful)
// - Referrer-Policy
// - Permissions-Policy
```

**Recommendation:** Add additional security headers.

#### 5. **WebSocket Compression Risk** ⚠️

```go
upg.EnableCompression = s.cfg.EnableCompression
```

**Issue:** WebSocket compression can be vulnerable to BREACH-like attacks if sensitive data is compressed.

**Recommendation:** Document this risk or consider disabling compression for authenticated sessions.

#### 6. **No TLS Configuration** ℹ️

The application doesn't appear to support HTTPS directly (relies on reverse proxy).

**Recommendation:** Consider adding optional TLS support for direct HTTPS deployment.

#### 7. **Log Information Leakage** ⚠️

```go
// In web/server.go
log.Printf("[API] %s %s %s %d %v", clientIP, r.Method, r.URL.Path, sw.status, duration)
```

**Issue:** Logs client IPs which could be sensitive in some environments.

**Recommendation:** Make IP logging configurable or anonymize.

---

## 📋 SUMMARY TABLE

| Category | Score | Key Findings |
|----------|-------|--------------|
| **Code Quality** | ⭐⭐⭐⭐☆ (4/5) | Clean architecture, good documentation, minor race conditions |
| **Performance** | ⭐⭐⭐⭐☆ (4/5) | Efficient ring buffer, good caching, some optimization opportunities |
| **Security** | ⭐⭐⭐⭐☆ (4/5) | Strong fundamentals (Landlock, Argon2id), TrustProxy needs hardening |

---

## 🎯 PRIORITY RECOMMENDATIONS

| Priority | Issue | Impact | Effort |
|----------|-------|--------|--------|
| 🔴 **P1** | TrustProxy IP validation | High | Low |
| 🔴 **P1** | Storage directory permissions (0700) | Medium | Low |
| 🟡 **P2** | Argon2 time cost increase | Medium | Low |
| 🟡 **P2** | WebSocket client.paused race condition | Medium | Low |
| 🟡 **P2** | Add security headers (HSTS, etc.) | Low | Low |
| 🟢 **P3** | Input validation for API parameters | Medium | Medium |
| 🟢 **P3** | Reduce magic numbers | Low | Medium |
| 🟢 **P3** | Pre-allocate slices in hot paths | Low | Low |

---

## 🏆 Overall Assessment

**Kula is a well-engineered monitoring tool** with strong security fundamentals and thoughtful architecture. The codebase demonstrates good Go practices, and the security measures (Landlock sandbox, Argon2id, CSP) show security-conscious development.

**Key strengths:**
- Zero-dependency single binary deployment
- Landlock process sandboxing
- Tiered ring-buffer storage with bounded memory usage
- Comprehensive authentication system

**Main areas for improvement:**
1. Harden TrustProxy configuration
2. Fix minor race conditions
3. Add more security headers
4. Increase password hashing parameters

The project is **production-ready** for internal/controlled environments but would benefit from the P1 security hardening before deployment in high-security contexts.
