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
