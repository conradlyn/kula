# Building & Toolchain

## Prerequisites

- **Go** — the version pinned in [go.mod](../../go.mod) (`go 1.26.4` at time of writing).
- Optional dev tools used by `check.sh`:
  - [`govulncheck`](https://golang.org/x/vuln/cmd/govulncheck)
  - [`golangci-lint`](https://golangci-lint.run/)

The binary is **CGO-free** (`CGO_ENABLED=0`) and fully static.

## Quick builds

```bash
# Dev build (~20 MB, with symbols)
CGO_ENABLED=0 go build -o kula ./cmd/kula/

# Production build (~14 MB, ~4 MB xz-compressed)
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -buildvcs=false -o kula ./cmd/kula/
```

## The build script

[`addons/build.sh`](../../addons/build.sh) wraps the production build and reads the version
from the [`VERSION`](../../VERSION) file:

```bash
./addons/build.sh          # current architecture only
./addons/build.sh cross    # cross-compile amd64, arm64, riscv64 → dist/
./addons/build.sh --help
```

Cross builds use:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=<arch> go build \
  -trimpath -ldflags="-s -w" -buildvcs=false \
  -o dist/kula-linux-<version>-<arch> ./cmd/kula/
```

Supported targets: `linux/amd64`, `linux/arm64`, `linux/riscv64`.

## The check script

[`addons/check.sh`](../../addons/check.sh) is the canonical pre-commit gate. It runs, in order:

1. **`govulncheck ./...`** — known-vulnerability scan (prefers `~/go/bin`, then system; skips
   with a hint if absent).
2. **`go vet ./...`**.
3. **`go test -v -race ./...`** — full test suite with the race detector.
4. **`golangci-lint run ./...`** — linting (skips with a hint if absent).

```bash
./addons/check.sh
```

> All four checks must pass before merging. The
> [AGENTS.md](../../AGENTS.md) rules name this script as the test suite.

## Tests, fuzzing & benchmarks

```bash
# Unit tests with the race detector
go test -race ./...

# Storage benchmark suite (default 3s per bench, pretty output)
./addons/benchmark.sh

# Fuzz targets
./addons/fuzz.sh
```

See [Testing & QA](12-testing.md) for what's covered.

## Updating dependencies

```bash
./addons/go_modules_updates.py   # bump used modules to latest minor/patch
go get -u ./...
go mod tidy
```

There are companion scripts: [`addons/chartjs-updates.py`](../../addons/chartjs-updates.py)
(refresh the bundled Chart.js), and [`addons/update.py`](../../addons/update.py).

## Python helper linting

The Python operator scripts are formatted/linted strictly:

```bash
black addons/*.py
pylint addons/*.py
mypy --strict addons/*.py
```

## Packaging

Distro and container package builders live in `addons/` — see [Packaging & Release](15-packaging.md):

```bash
./addons/build_deb.sh      # → dist/kula-*.deb
./addons/build_rpm.sh      # → dist/kula-*.rpm
./addons/build_aur.sh      # → dist/aur (then makepkg -si)
./addons/build_snap.sh     # → dist/kula-*.snap (needs snapcraft + LXD)
./addons/docker/build.sh   # Docker image
```

## CI

GitHub Actions workflows live in [`.github/workflows/`](../../.github/workflows/):

- `ci.yml` — build + the `check.sh`-style verification.
- `semgrep.yml` — static analysis security scan.

Next: [Collector Subsystem](04-collector.md).
