**Audit Summary**

| Area | Score (0–10) | Severity | Remarks |
| --- | --- | --- | --- |
| Go — HTTP/Web/API | 8.5 | Low | Strong template/response hardening, structured auth, CSRF and CSP nonces, Landlock sandbox. |
| Go — Storage/Auth/Collector | 9.0 | Low | Well-structured ring-buffer storage with data validation & migration; Argon2 hashing and secure session handling. |
| JavaScript/HTML | 8.0 | Medium | CSP + SRI supported; DOM manipulations sanitized in most places but some raw innerHTML usage (see below). |

---

## Findings

### 1. **Gin/Go HTTP surface & templates**
- **Severity:** Low  
- **Score impact:** -0.5  
- **Details:** CSP (`script-src 'self' 'nonce-...'`) + nonce injected ([securityMiddleware](cci:1://file:///home/c0m4r/.windsurf/worktrees/kula/kula-bf9deaa6/internal/web/server.go:163:0-180:1)) and `template` rendering use `html/template` to auto-escape. Static handler normalizes `staticFS` access and forbids directory listing, while [calculateSRIs](cci:1://file:///home/c0m4r/.windsurf/worktrees/kula/kula-bf9deaa6/internal/web/server.go:638:0-662:1) and template helpers guard script loading. TLS detection correctly sets `Secure` cookies on login/logout.  
- **Recommendation:** None; already best practices.

### 2. **Authentication/session handling**
- **Severity:** Low  
- **Score impact:** -0.2  
- **Details:** Login rate limiter, Argon2 hash comparison (`subtle.ConstantTimeCompare`), session tokens hashed before storage, sliding expiration, CSRF ([ValidateOrigin](cci:1://file:///home/c0m4r/.windsurf/worktrees/kula/kula-bf9deaa6/internal/web/auth.go:305:0-323:1)), auth middleware supporting cookie + bearer.  
- **Recommendation:** Ensure [ValidateOrigin](cci:1://file:///home/c0m4r/.windsurf/worktrees/kula/kula-bf9deaa6/internal/web/auth.go:305:0-323:1) permits legitimate requests behind proxies (already conditionally trusts proxy). Possibly log origin mismatches for monitoring.

### 3. **Storage & collector subsystem**
- **Severity:** Low  
- **Score impact:** -0.0  
- **Details:** Tiered ring buffer, header validation, migration path, query caching, safe downsampling. Disk writes validated for max size and double-checks corrupted headers.  
- **Recommendation:** Consider additional file locking when multiple processes may access same directory (if ever required). Already uses `sync` to guard in-process concurrency.

### 4. **JavaScript DOM updates (`innerHTML`)**
- **Severity:** Medium  
- **Score impact:** -1.0  
- **Details:** [header.js](cci:7://file:///home/c0m4r/.windsurf/worktrees/kula/kula-bf9deaa6/internal/web/static/js/app/header.js:0:0-0:0) builds `sysInfo` strings that may include user/system-controlled values (`clock_source`, `entropy`, `user_count`). [alerts.js](cci:7://file:///home/c0m4r/.windsurf/worktrees/kula/kula-bf9deaa6/internal/web/static/js/app/alerts.js:0:0-0:0) builds alert dropdown HTML using `state.alerts` but safely escapes titles/details via shared [escapeHTML](cci:1://file:///home/c0m4r/.windsurf/worktrees/kula/kula-bf9deaa6/internal/web/static/js/app/state.js:124:0-125:143). [header.js](cci:7://file:///home/c0m4r/.windsurf/worktrees/kula/kula-bf9deaa6/internal/web/static/js/app/header.js:0:0-0:0) also uses `innerHTML` but builds strings from sanitized data and mostly static text. However, `s.sys.clock_source`, `s.sys.entropy`, etc., come from backend metrics (trusted but still external).  
- **Recommendation:** Prefer templating DOM updates or set `textContent`/`dataset` rather than raw `innerHTML`. If `innerHTML` persists, continue escaping via [escapeHTML](cci:1://file:///home/c0m4r/.windsurf/worktrees/kula/kula-bf9deaa6/internal/web/static/js/app/state.js:124:0-125:143) (already used for `clock_source`/`entropy`/`user_count`), and extend to all dynamic pieces (e.g., `sysInfo` array currently includes `s.self` string without escaping; ensure metrics can't inject characters by escaping all inserted values). 

### 5. **JavaScript event wiring & WebSocket flow**
- **Severity:** Low  
- **Score impact:** -0.3  
- **Details:** WebSocket reconnect & queueing limit message size (1 MB) and sets read/write deadlines, per-IP/global connection caps. Front-end handles buffering, reconnect delays, and history fetch gating.  
- **Recommendation:** Consider capping queue max length more aggressively (already 120). Optionally log oversized messages.

### 6. **Prometheus endpoint**
- **Severity:** Low  
- **Score impact:** -0.1  
- **Details:** Optional bearer token, consistent label escaping. No direct security concerns.  
- **Recommendation:** Document in config that metrics should remain behind auth when enabled.

---

## Recommendations

1. **Sanitize all `innerHTML` usage** (medium). Ensure all interpolated values (even from trusted collectors) pass through [escapeHTML](cci:1://file:///home/c0m4r/.windsurf/worktrees/kula/kula-bf9deaa6/internal/web/static/js/app/state.js:124:0-125:143). E.g., wrap `s.self` data in [escapeHTML](cci:1://file:///home/c0m4r/.windsurf/worktrees/kula/kula-bf9deaa6/internal/web/static/js/app/state.js:124:0-125:143) when building `sysInfo` in [header.js](cci:7://file:///home/c0m4r/.windsurf/worktrees/kula/kula-bf9deaa6/internal/web/static/js/app/header.js:0:0-0:0) or avoid `innerHTML` altogether by creating DOM nodes.
2. **Monitor auth origin failures** (low). Log invalid origin attempts in [ValidateOrigin](cci:1://file:///home/c0m4r/.windsurf/worktrees/kula/kula-bf9deaa6/internal/web/auth.go:305:0-323:1) to spot possible attack attempts.
3. **Consider optional SRI header** (low). Already computes SRI for script tags; bundling `Content-Security-Policy-Report-Only` or `Subresource Integrity` headers would provide defense-in-depth for future static hosting changes.

---

## Severity Legend
- **Low:** Controls exist though minor improvements suggested.
- **Medium:** Minor insecure pattern (e.g., DOM insertion) that should be tightened.
- **High:** (None found) no major vulnerabilities identified.

---

## Overall Summary  
The project shows strong awareness of web security: CSP with nonces, SRI, CSRF origin checks, secure cookies, session hashing, Landlock sandboxing, and rate limiting. Storage and collector paths include safeguards and efficient caching. The main weakness resides in frontend DOM updates that rely on `innerHTML`; hardening these (or ensuring all interpolated values are escaped) will eliminate the remaining attack surface. Overall score: **8.8/10**.
