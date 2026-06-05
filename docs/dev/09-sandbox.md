# Landlock Sandbox

File: [`internal/sandbox/sandbox.go`](../../internal/sandbox/sandbox.go) (introduced in v0.4.0)

Kula confines itself at runtime using the **Linux Landlock LSM** via the
[`go-landlock`](https://github.com/landlock-lsm/go-landlock) library. After startup, the process
restricts its own filesystem and network access to the minimum it needs — so even a
hypothetical code-execution bug can't read arbitrary files or open arbitrary network
connections.

This is enforced **in-process**, independent of (and complementary to) the hardened systemd unit
([Service Management](../user/15-service-management.md)).

## When it's applied

`runServe` calls `sandbox.Enforce(configPath, storageDir, webCfg, appsCfg, ollamaCfg)` early,
*before* the collection loop starts:

```go
if err := sandbox.Enforce(configPath, cfg.Storage.Directory, cfg.Web,
                          cfg.Applications, cfg.Ollama); err != nil {
    log.Printf("Warning: Landlock sandbox not enforced: %v", err)
}
```

It is **non-fatal**: on unsupported kernels the warning is logged and Kula runs unconfined.

## Kernel requirements & graceful degradation

- Requires **kernel 5.13+** with Landlock enabled.
- Kula checks the Landlock **ABI version** at startup and uses `BestEffort()` so it degrades
  gracefully on older kernels (applying whatever subset is available, or nothing).

## Filesystem rules

| Path | Access |
|------|--------|
| `/proc` | read-only |
| `/sys` | read-only |
| config file | read-only |
| storage directory | read-write |
| `/etc/hosts`, `/etc/resolv.conf`, `/etc/nsswitch.conf` | read-only (for DNS) |

The read-write grant on the storage directory is what lets the storage engine, the custom-metrics
socket, and the backup writer function under confinement.

## Network rules

- **TCP bind** is allowed only on the configured web port (so the server can listen).
- **TCP connect** is allowed only to the ports of the **enabled** application collectors —
  conditionally added for nginx, Apache2, MySQL, PostgreSQL, and Ollama. The port is parsed from
  each module's configured URL/host (defaulting to 80, or 443 for `https`).

Because the connect rules are derived from your config at startup, enabling an application or
changing its port automatically updates the allowed set — no manual sandbox tweaking. Each
applied rule is logged (e.g. `foo:connect-tcp/9000`).

## Adding a connect rule for a new collector

If you add an application collector that makes outbound HTTP/DB connections, you must extend the
sandbox so it can reach that port. The pattern (parse the port from the configured URL, append a
`landlock.ConnectTCP(port)` rule when the module is enabled) is shown step-by-step in
[Adding a Metric Type](14-adding-metrics.md#5-sandbox).

## Tests

[`sandbox_test.go`](../../internal/sandbox/sandbox_test.go) asserts that, once enforced:

- Writing outside the storage directory fails.
- Executing outside the allowed paths fails.
- Dialing an external (non-allowed) network address fails.

These are the negative tests proving the confinement actually holds.

Next: [Frontend](10-frontend.md).
