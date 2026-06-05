# Testing & QA

Kula has unit tests, race tests, native fuzz targets, benchmarks, in-process runtime security
tests, and an out-of-tree black-box scanner. The canonical gate is `./addons/check.sh`.

## The check gate

```bash
./addons/check.sh
```

Runs, in order (all must pass):

1. `govulncheck ./...` — known-vulnerability scan.
2. `go vet ./...`.
3. `go test -v -race ./...` — full suite with the race detector.
4. `golangci-lint run ./...`.

CI runs the equivalent ([`.github/workflows/ci.yml`](../../.github/workflows/ci.yml)) plus
Semgrep ([`semgrep.yml`](../../.github/workflows/semgrep.yml)).

## Unit tests

Notable suites:

| Area | Tests |
|------|-------|
| Collectors | `cpu_test.go`, `disk_test.go`, `network_test.go`, `memory_test.go`, `process_test.go`, `system_test.go`, `containers_test.go`, `app_test.go` |
| Storage | `store_test.go`, `tier_test.go`, `codec_test.go`, `snapshot_test.go`, `migration_test.go` |
| Web/security | `auth_test.go`, `server_test.go`, `websocket_test.go`, `ollama_test.go`, `prometheus_test.go`, `runtime_security_test.go` |
| Config | `config_test.go` |
| Sandbox | `sandbox_test.go` |
| Backup | `backup_test.go`, `cron_test.go` |
| i18n | `i18n_test.go` |
| TUI | `tui_test.go` |

Collectors are driven against synthetic `/proc` and `/sys` fixture trees under
[`internal/collector/testdata/`](../../internal/collector/testdata/), so they run deterministically
on any machine. The sandbox tests are *negative* tests — they assert that forbidden
writes/exec/network actually fail once Landlock is enforced.

## Fuzzing

Go-native fuzz targets (each with a committed seed corpus that also runs under plain
`go test`):

| Package | Target |
|---------|--------|
| `config` | `FuzzNormalizeBasePath`, `FuzzValidateOllamaURL`, `FuzzParseSize` |
| `web` | `FuzzValidateOrigin`, `FuzzGetClientIP` |
| `collector` | `FuzzParseNginxStatus`, `FuzzParseApache2Status`, `FuzzParseUintBytes`, `FuzzCustomMessage`, `FuzzCollectProcessesStat` |
| `storage` | `FuzzDecodeSample`, `FuzzExtractTimestamp`, `FuzzEncodeDecodeRoundTrip` |

Run mutation-based fuzzing across all targets:

```bash
./addons/fuzz.sh            # 30s per target, all targets
./addons/fuzz.sh 2m         # 2 minutes per target
./addons/fuzz.sh 1m Decode  # only targets matching "Decode"
./addons/fuzz.sh -r 1m      # with the race detector
./addons/fuzz.sh -l         # list discovered targets
```

The decode fuzzers are particularly important: the codec must **never panic** on malformed
on-disk records. Crashers are saved under `testdata/fuzz/`.

## Benchmarks

The storage engine has a benchmark suite:

```bash
./addons/benchmark.sh                 # 3s per bench, single pass, pretty output
./addons/benchmark.sh 5s              # longer for tighter numbers
./addons/benchmark.sh -c 5 -o new.txt # 5 runs, benchstat-compatible output
benchstat old.txt new.txt             # compare two runs
```

Generate large multi-day datasets to benchmark realistic tier rollups and wrap behavior with
[`cmd/gen-mock-data`](../../cmd/gen-mock-data/main.go).

## Runtime security tests

[`internal/web/runtime_security_test.go`](../../internal/web/runtime_security_test.go) spins up a
real server in-process and probes it (e.g. raw-socket path traversal) — verifying defenses end to
end, not just unit logic.

## Black-box scanning (kula-scan)

[`kula-scan`](13-kula-scan.md) verifies the same defenses **in the field**, over HTTP/WebSocket,
against a running instance — complementing the in-tree tests. It imports nothing from `internal/`,
so every assertion is made over the wire. Use it as a release/CI gate against a stood-up
instance.

## Python helpers

The operator/build Python scripts are checked with:

```bash
black addons/*.py
pylint addons/*.py
mypy --strict addons/*.py
```

Next: [kula-scan](13-kula-scan.md).
