## KULA Codebase Security & Quality Audit

### Overall Scores
| Area | Score (0–10) | Summary |
| --- | --- | --- |
| Security | **7.2** | Strong crypto defaults (Argon2id, hashed sessions, CSP, CSRF middleware) and defensive HTTP stack, but a few gaps (missing persistent login attempt cleanup, lax transport protections, unbounded JSON decode) leave hardening opportunities. |
| Code Quality | **7.8** | Generally idiomatic Go/JS with tests for auth, storage, and collectors. Some duplication and lack of centralized error handling; config parsing is clear, but JS could benefit from modularization. |
| Performance | **7.0** | Efficient collectors and websockets with back-pressure and gzip option. However, sessions/rate limiter kept solely in-memory, JSON decoding without size limits in some handlers, and potential Chart.js heavy loads can impact memory/CPU under stress. |

---

### Findings

#### 1. Missing Transport Enforcement for Auth Cookies (Security, **High**)
*File:* [internal/web/server.go](cci:7://file:///home/c0m4r/.windsurf/worktrees/kula/kula-c1001632/internal/web/server.go:0:0-0:0) @ lines 517-525  
Cookies are marked `Secure` only when TLS is detected or `X-Forwarded-Proto` is `https`. When running behind a proxy without TLS termination, an attacker on the same network could sniff session cookies.  
**Recommendation:** Add config to force `Secure` and `SameSite=Strict` regardless of detection, or refuse to start when `Auth.Enabled` but transport isn’t HTTPS/trusted proxy. Document deployment constraints clearly.

#### 2. Rate Limiter Data Retention / DoS Vector (Security, **Medium**)  
*File:* [internal/web/auth.go](cci:7://file:///home/c0m4r/.windsurf/worktrees/kula/kula-c1001632/internal/web/auth.go:0:0-0:0) RateLimiter (@68-90, 218-234)  
Attempts map grows per attacking IP until cleanup runs (only every 5 min in server goroutine). Sustained attacks from many IPs could exhaust memory.  
**Recommendation:** Evict entries eagerly (e.g., use ring buffer capped per IP) or limit map size with LRU.

#### 3. Unbounded JSON Decode in Several Handlers (Security/Perf, **Medium**)  
*File:* [internal/web/server.go](cci:7://file:///home/c0m4r/.windsurf/worktrees/kula/kula-c1001632/internal/web/server.go:0:0-0:0)  
Only [handleLogin](cci:1://file:///home/c0m4r/.windsurf/worktrees/kula/kula-c1001632/internal/web/static/js/app/auth.js:100:0-127:1) wraps body with `http.MaxBytesReader`; other POST/PUT handlers (e.g., `/api/history`, `/api/config`) accept unlimited JSON and could be abused to allocate memory.  
**Recommendation:** Apply `MaxBytesReader` and explicit size checks on all endpoints accepting request bodies.

#### 4. Session Store Not Persisted Between Restarts in Default Config (Security, **Medium**)  
*Files:* [internal/web/auth.go](cci:7://file:///home/c0m4r/.windsurf/worktrees/kula/kula-c1001632/internal/web/auth.go:0:0-0:0) ([SaveSessions](cci:1://file:///home/c0m4r/.windsurf/worktrees/kula/kula-c1001632/internal/web/auth.go:270:0-295:1), [LoadSessions](cci:1://file:///home/c0m4r/.windsurf/worktrees/kula/kula-c1001632/internal/web/auth.go:236:0-268:1)) and [cmd/kula/main.go](cci:7://file:///home/c0m4r/.windsurf/worktrees/kula/kula-c1001632/cmd/kula/main.go:0:0-0:0) (not read but default storageDir empty). If `storageDir` is empty (as when server is run without persistence), sessions disappear on restart leading to forced logout but not security issue. More concern: when `storageDir` exists, `sessions.json` is stored without encryption; compromise of disk reveals hashed tokens (SHA-256 only).  
**Recommendation:** Document storage requirements; consider encrypting session storage or using OS keyring. At minimum, ensure storageDir defaults to secure location.

#### 5. WebSocket Origin Check Allows Empty Origin (Security, **Medium**)  
*File:* [internal/web/websocket.go](cci:7://file:///home/c0m4r/.windsurf/worktrees/kula/kula-c1001632/internal/web/websocket.go:0:0-0:0) @24-47  
Allowing empty Origin helps CLI clients but opens CSRF-like CSWSH from browsers that can omit Origin via sandboxed iframes or extension contexts.  
**Recommendation:** Provide config flag to disallow empty Origin by default; require explicit opt-in for CLI compatibility.

#### 6. Lack of CSRF Protection on Login Endpoint (Security, **Low**)  
[handleLogin](cci:1://file:///home/c0m4r/.windsurf/worktrees/kula/kula-c1001632/internal/web/static/js/app/auth.js:100:0-127:1) is exempt from [AuthMiddleware](cci:1://file:///home/c0m4r/.windsurf/worktrees/kula/kula-c1001632/internal/web/auth.go:176:0-203:1) but protected by CSRF middleware via global mux; however, [ValidateOrigin](cci:1://file:///home/c0m4r/.windsurf/worktrees/kula/kula-c1001632/internal/web/auth.go:305:0-323:1) rejects requests without Origin/Referer, which blocks CLI/API clients unintentionally and may lead admins to disable CSRF entirely.  
**Recommendation:** Provide token-based CSRF for login or relax requirement only for `/api/login` with rate limiting; document expected headers.

#### 7. Potential Information Leakage in `/api/config` (Security, **Low**)  
API returns kernel, OS, version info even when auth disabled ([checkAuth](cci:1://file:///home/c0m4r/.windsurf/worktrees/kula/kula-c1001632/internal/web/static/js/app/auth.js:5:0-29:1) fetches config before ensuring login). Attackers can fingerprint deployments remotely.  
**Recommendation:** Gate `/api/config` behind auth when `Auth.Enabled`; when auth disabled, return limited fields configurable by admin.

#### 8. JavaScript: Missing Content-Length Checks on Fetch Responses (Quality, **Low**)  
Front-end blindly `r.json()`. If server compromised and returns massive payload, browser memory blowup. Not critical but worth adding guard to avoid UI lockups.

#### 9. Configuration Hard-Coded Defaults (Security, **Low**)  
[DefaultConfig](cci:1://file:///home/c0m4r/.windsurf/worktrees/kula/kula-c1001632/internal/config/config.go:120:0-173:1) sets Argon2 parameters but leaves username/password empty (auth disabled). Running with defaults exposes dashboard publicly.  
**Recommendation:** Emit startup warning when auth disabled, require explicit `enabled: false` flag acknowledgement, or auto-generate random credentials.

#### 10. Metrics Endpoint Token Optional (Security, **Low**)  
Prometheus endpoint can be left unauthenticated; log warns but still exposes infrastructure metrics.  
**Recommendation:** Provide rate limiting and optional IP whitelist; consider disabling by default unless token configured.

---

### Recommendations Summary
1. **Transport Hardening:** Require HTTPS for auth-enabled deployments; add strict cookie flags and config validation.
2. **Request Size Limits:** Use `http.MaxBytesReader` or manual checks across all JSON endpoints.
3. **Rate Limiter Improvements:** Implement bounded data structures and faster cleanup cadence.
4. **WebSocket Origin Policy:** Add configuration to enforce Origin even when absent.
5. **Config & Metrics Exposure:** Restrict sensitive info when auth disabled; enforce auth on `/metrics` unless explicitly allowed.
6. **Documentation:** Highlight secure deployment practices (proxy expectations, session storage path permissions, CLI limitations under CSRF).

---

### Overall Summary
KULA demonstrates thoughtful security primitives—hashed sessions, Argon2id, CSP with dynamic nonces, CSRF middleware, logging, and rate limiting. The Go backend is mostly idiomatic and test-covered; the TUI/web stack uses clear modular JS. However, a few operational assumptions (trusted TLS termination, manual config of auth, unlimited request sizes) could lead to exposure if admins deploy with defaults or on hostile networks. Addressing the identified gaps will push the platform toward a hardened, production-ready monitoring solution.
