# Kula Security Audit & Code Review Report

**Date:** 2026-03-20  
**Project:** [c0m4r/kula](https://github.com/c0m4r/kula)  
**Auditor:** Antigravity (Professional Security Researcher & Programmer)

---

## 1. Executive Summary

Kula is a lightweight, self-contained server monitoring tool written in Go with a modern web frontend. This audit focused on security architecture, code quality, and performance.

The overall security level is **High**. The project demonstrates advanced security practices rarely seen in similar monitoring tools, particularly the use of **Landlock-based process sandboxing** and **Content Security Policy (CSP) with dynamic nonces**.

### Overall Scores (0-10)
- **Security:** 9.5/10
- **Code Quality:** 9.0/10
- **Performance:** 8.5/10

---

## 2. Security Analysis

### 2.1. Process Sandboxing (Landlock)
Kula implements **Landlock LSM** to restrict its own process at the kernel level.
- **Severity:** [CRITICAL STRENGTH]
- **Mechanism:** [sandbox.go](file:///home/c0m4r/ai/kula/internal/sandbox/sandbox.go)
- **Details:** The process is restricted to read-only access for `/proc` and `/sys`, and read-write only for its dedicated storage directory. It is also restricted to `TCP_BIND` for its configured port, preventing unwanted outbound connections or binding to other ports if compromised.

### 2.2. Authentication & Session Management
- **Password Hashing:** Uses **Argon2id**, which is the current state-of-the-art for password hashing.
- **Session Security:** Sessions are **hashed** before being stored on disk or looked up in memory. This prevents an attacker with filesystem access from obtaining valid session tokens.
- **CSP & XSS:** Implements a strict Content Security Policy with **cryptographic nonces** generated per-request. All inline scripts (if any) and external scripts require this nonce.
- **SRI:** Subresource Integrity hashes are calculated at startup and pinned to all script tags in the HTML templates.

### 2.3. Potential Weaknesses & Recommendations

#### [LOW] IP Spoofing Risk via TrustProxy
- **Location:** [server.go:666](file:///home/c0m4r/ai/kula/internal/web/server.go#L666)
- **Issue:** The `getClientIP` function trusts the *last* IP in `X-Forwarded-For`. If Kula is behind multiple proxies or a proxy that doesn't scrub `X-Forwarded-For`, an attacker could append their own IP to spoof the client address.
- **Recommendation:** Implement a more robust `TrustProxy` logic that allows specifying trusted CIDRs and iterates through the `X-Forwarded-For` list from right-to-left.

#### [LOW] Informational Leakage in API Errors
- **Location:** [server.go:153](file:///home/c0m4r/ai/kula/internal/web/server.go#L153)
- **Issue:** `jsonError` often returns `err.Error()`. For database or filesystem errors, this might leak internal paths or structural details.
- **Recommendation:** Map internal errors to generic error messages for the API, while keeping detailed logs on the server side.

#### [INFO] CSRF Protection
- **Mechanism:** [auth.go:327](file:///home/c0m4r/ai/kula/internal/web/auth.go#L327)
- **Observation:** Currently uses `Origin` and `Referer` validation. While effective for modern browsers, it is less robust than synchronizer tokens. Given that Kula is mostly used via a browser, this is acceptable.

---

## 3. Code Quality Analysis

### 3.1. General Impression
The Go codebase is highly idiomatic, clean, and well-structured. It avoids "clever" hacks in favor of clarity and maintainability.

### 3.2. Automated Checks
- **Results:** 100% of the internal test suite passed. `golangci-lint` reported **zero issues** with default production-grade settings.
- **Refactoring Note:** The recent refactor from a monolithic `app.js` to modular ES6-like components in `js/app/` significantly improved frontend maintainability.

### 3.3. Recommendations
- **Testing:** The test coverage for `internal/collector` is good, but some hardware-specific paths (like NVIDIA GPU) rely on mocks. Ensure real-world validation on diverse hardware.
- **Redundancy:** Minor redundant `Flush()` calls found in `Tier.Close()`, though harmless.

---

## 4. Performance Analysis

### 4.1. Storage Architecture
- **Mechanism:** [store.go](file:///home/c0m4r/ai/kula/internal/storage/store.go)
- **Details:** The tiered storage system is highly efficient. By aggregating data into 1m and 1h resolutions, Kula maintains long-term history with fixed-size ring buffers, preventing disk bloat.
- **Memory Efficiency:** Use of `sync.Pool` for binary encoding/decoding buffers significantly reduces GC pressure on the hot `1 Hz` collection path.

### 4.2. Bottlenecks
- **Process Collection:** [process.go:10](file:///home/c0m4r/ai/kula/internal/collector/process.go#L10)
- **Observation:** On systems with extremely high thread counts or process churn, the current `collectProcesses` (which iterates `/proc` and every `/task` directory) may consume noticeable CPU.
- **Recommendation:** Consider adding an option to disable thread-count collection or increase the collection interval for the process tab.

---

## 5. Audit Conclusion

**Final Verdict: SECURE**

Kula is a model for how a modern, self-contained system tool should be built. Its developers have prioritized security at every layer, from kernel-level sandboxing to frontend script integrity. 

### Severity Labels Summary
| Severity | Count | Summary |
|---|---|---|
| **CRITICAL** | 0 | No critical vulnerabilities found. |
| **HIGH** | 0 | No high-severity issues found. |
| **MEDIUM** | 0 | No medium-severity issues found. |
| **LOW** | 2 | Minor IP spoofing risk and potential error leakage. |
| **INFO** | 2 | CSRF mechanism and performance trade-offs. |

### Final Recommendations
1.  **Refine IP extraction** to handle complex proxy chains.
2.  **Sanitize API error messages** to avoid leaking internal system details.
3.  **Continue the audit** as new collectors (e.g., eBPF or specialized hardware) are added.
