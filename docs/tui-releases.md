# TUI release preparation

Maintainers need Go 1.27.3, Python 3.9 or later, and the normal repository tools.
The modules in `go.mod`/`go.sum` are pinned; builds use `-mod=readonly`,
`CGO_ENABLED=0`, and need no C cross-compiler.

```sh
go test ./...
go vet ./...
make verify
make release-tui VERSION=v0.1.0
```

Use a new output directory for every build. `dist/tui/v0.1.0/` contains ZIPs for
macOS and Linux on amd64 and arm64, `SHA256SUMS`, and `release.json`. Each ZIP
contains only `agent-toolkit` and `metadata.json`; metadata records the release,
target, and managed backend protocol. `release.json` additionally records both
archive and executable SHA-256 hashes for the launcher. Review and pin these
locally generated hashes in source; never trust checksums fetched alongside an
untrusted download as the expected values.

Inspect all archives and compare their hashes with `SHA256SUMS`. Execute
`agent-toolkit --version` and exercise the UI on each available target. Record
actual runtime results separately; cross-compilation alone proves only build
support. The development version prints `development`; release builds embed the
requested version.

GitHub release publication is a separate maintainer action. Upload the four
archives, checksum list, and metadata to the matching version tag. Download and
verify every uploaded archive before installing a pin or enabling interactive
routing. Do not describe a locally built version as publicly available.

## Verified launcher

`python3 scripts/launch_tui.py` launches a published binary without Go. It needs
Python 3.9 or later, HTTPS access on the first run, and the Bash backend's normal
prerequisites. It always uses the backend in this checkout. There is currently
no published release pin, so this command reports the Bash fallback. `make
install` continues using Bash until a published pin is validated and interactive
routing is enabled separately.

After publication, review the downloaded archives against the locally generated
hashes, then copy the reviewed `release.json` to `scripts/tui-release.json` in a
source change. The pin must name all four targets and include archive and binary
hashes. The launcher downloads only the pinned version from this repository's
GitHub Releases; it never resolves a latest version or builds from source.

Verified executables are cached under
`${XDG_CACHE_HOME:-$HOME/.cache}/agent-toolkit-tui/<version>-<target>/`.
The launcher requires private, owned cache directories, serializes launches with
a per-target lock, publishes downloads atomically, and verifies the executable
hash on every run. A valid cache works offline. ZIP members and metadata are
checked before publication; failed downloads and extraction leave no executable.
The backend's read-only protocol check runs before download or launch.

Use `python3 scripts/launch_tui.py --version` to verify a host launch. If a cache
is corrupt, remove only that version/target directory while no launcher is
running, then retry. On download, compatibility, or platform failure, use
`bash scripts/install.sh` with the existing CLI options. Launcher failures do
not apply installation changes. Publication and interactive routing remain
separate from launcher fixture validation.
