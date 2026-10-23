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
no published release pin, so this command reports the Bash fallback. Terminal
`make install` uses Bash until a published pin is installed. Once a pin exists,
the entrypoint runs the launcher; its protocol, archive, and executable checks
must pass before the TUI starts. Failed checks stop setup without falling
through to a Bash installation. `make install-cli`, direct `scripts/install.sh`
calls, explicit `scripts/setup.sh` arguments, and non-terminal `make install`
always use Bash without downloading.

After publication, review the downloaded archives against the locally generated
hashes, then copy the reviewed `release.json` to `scripts/tui-release.json` in a
source change. This activates terminal routing; no separate toggle is needed.
Validate fresh and cached terminal `make install`, non-terminal setup, and
`make install-cli` against the published assets before recording release
activation complete. The pin must name all four targets and include archive
and binary hashes. The launcher downloads only the pinned version from this
repository's GitHub Releases; it never resolves a latest version or builds
from source.

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
not apply installation changes. Removing the pin restores terminal Bash routing;
pinning a prior compatible release rolls back the TUI version. Neither changes
the installed skill selection by itself. Publication and verification against
public assets remain separate from local fixture validation.
