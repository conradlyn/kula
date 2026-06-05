# Packaging & Release

All packaging is driven by Bash scripts in [`addons/`](../../addons/). Output goes to `dist/`.
The version comes from the [`VERSION`](../../VERSION) file, so the release flow is "bump VERSION,
run the builders."

## Release orchestration

[`addons/release.sh`](../../addons/release.sh) builds the full release set:

1. Reads `VERSION`.
2. Wipes `dist/` (so checksums reflect exactly this release).
3. Cross-compiles all architectures (`./build.sh cross` → amd64, arm64, riscv64).
4. For each binary, assembles a `kula/` bundle containing the binary plus `CHANGELOG.md`,
   `VERSION`, `LICENSE`, `README.md`, `config.example.yaml`, `scripts/`, `bash-completion`,
   `init/`, and `man/`, and produces both a `.tar.gz` and a gzipped single binary.
5. Generates `CHECKSUMS.sha256.txt` over the artifacts (used by the guided installer).

## Per-format builders

| Script | Output | Notes |
|--------|--------|-------|
| [`build.sh`](../../addons/build.sh) | raw binary / `dist/kula-linux-<ver>-<arch>` | `./build.sh cross` for all arches |
| [`build_deb.sh`](../../addons/build_deb.sh) | `dist/kula-*.deb` | installs systemd unit + `kula` user |
| [`build_rpm.sh`](../../addons/build_rpm.sh) | `dist/kula-*.rpm` | RHEL/Fedora family |
| [`build_aur.sh`](../../addons/build_aur.sh) | `dist/aur/` | then `makepkg -si` |
| [`build_snap.sh`](../../addons/build_snap.sh) | `dist/kula-*.snap` | needs snapcraft + LXD; `cross` for multi-arch |
| [`build_appimage.sh`](../../addons/build_appimage.sh) | AppImage | portable single-file |
| [`docker/build.sh`](../../addons/docker/build.sh) | Docker image | multi-arch via buildx |

### Examples

```bash
./addons/build_deb.sh && ls -1 dist/kula-*.deb
./addons/build_rpm.sh && ls -1 dist/kula-*.rpm
./addons/build_aur.sh && (cd dist/aur && makepkg -si)
./addons/build_snap.sh            # host arch
./addons/build_snap.sh cross      # amd64/arm64/riscv64 locally
```

## Docker

The image is a two-stage build ([`addons/docker/Dockerfile`](../../addons/docker/Dockerfile)):

- **Builder:** pinned `golang:1.26.4` (by digest), `CGO_ENABLED=0`, multi-arch via
  `ARG TARGETARCH`, `-trimpath -ldflags="-s -w" -buildvcs=false`, building only `./cmd/kula/`.
- **Runtime:** pinned `alpine:3.21` (by digest) with an unprivileged `kula:kula` user.

```bash
./addons/docker/build.sh
docker compose -f addons/docker/docker-compose.yml up -d
./addons/docker/push.sh        # Docker Hub
./addons/docker/push_ghcr.sh   # GHCR
```

The image expects host PID/network and read-only `/proc` at runtime — see the user
[Installation → Docker](../user/02-installation.md#docker) section.

## Snap specifics

The snap uses a **strict sandbox**, so capabilities are limited by default and extended with
`snap connect`. The snap packaging lives under [`snap/`](../../snap/). See the
[Snap wiki page](https://github.com/c0m4r/kula/wiki/Snap).

## Ansible

A deployment role is provided under [`addons/ansible/`](../../addons/ansible/) for fleet
rollouts (`deploy.sh`, `kula.yaml`, the `roles/kula` role with a `config.yaml.j2` template).

## Install scripts

- [`addons/install_v2.sh`](../../addons/install_v2.sh) — current guided installer (auto-detects
  distro, verifies checksums; flags `-y`/`--yes`, `--skip-verify`).
- [`addons/install.sh`](../../addons/install.sh) — earlier multi-distro installer.

## Packaging helpers

[`addons/packaging/`](../../addons/packaging/) holds size-trimming helpers (`remove_fonts.sh`,
`remove_game.sh`) for constrained builds.

## Release checklist

1. Update [`CHANGELOG.md`](../../CHANGELOG.md).
2. Bump [`VERSION`](../../VERSION).
3. `./addons/check.sh` — all green.
4. `./addons/release.sh` — build all artifacts + checksums.
5. Build/push the Docker image and any distro packages needed.
6. Publish the GitHub release with `dist/` artifacts and `CHECKSUMS.sha256.txt`.
7. Update install/standalone checksums referenced in `README.md` if applicable.

Next: [Contributing](16-contributing.md).
