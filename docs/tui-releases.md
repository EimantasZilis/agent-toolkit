# TUI releases

The release build produces binaries for macOS and Linux on amd64 and arm64.
Users can run them without Go. Maintainers need Go 1.27.3 or later, Python 3.9
or later, and the repository's usual tools.

## Build and publish

1. Run checks and build a version:

   ```sh
   go test ./...
   go vet ./...
   make verify
   make release-tui VERSION=v0.1.0
   ```

   Use a new output directory for each build. Builds use pinned Go modules,
   `-mod=readonly`, and `CGO_ENABLED=0`; no C cross-compiler is needed.

2. Inspect `dist/tui/v0.1.0/`. It contains four ZIPs, `SHA256SUMS`, and
   `release.json`. Each ZIP contains `agent-toolkit` and `metadata.json`, which
   records the version, target, and backend protocol. `release.json` also
   records archive and executable hashes.

3. Compare the archives with `SHA256SUMS`. On each available target, run
   `agent-toolkit --version` and try the UI. Record which targets you ran.
   Release builds print the requested version; development builds print `development`.

4. Publish the four ZIPs, checksums, and metadata to the matching GitHub release
   tag. Download every uploaded archive and compare it with the local hashes.
   Use the locally generated hashes as the expected values, not checksums
   downloaded with the assets.

5. Copy the verified `release.json` to `scripts/tui-release.json` in a source
   change. This pin must include all four targets and their archive and
   executable hashes. It enables terminal `make install` to launch the TUI.

6. Test fresh and cached terminal `make install`, non-terminal setup, and
   `make install-cli` against the published release before marking it active.
   Local builds and fixture tests do not verify published assets.

## Verified launcher

`python3 scripts/launch_tui.py` runs the pinned release with this checkout's
Bash backend. It needs Python 3.9 or later, the backend prerequisites, and HTTPS
for the first download. It checks backend compatibility, archive contents, and
hashes before starting the TUI. It downloads only the pinned GitHub release;
it never chooses the latest version or builds from source.

This checkout has no published pin yet, so `make install` uses Bash. After a
pin is added, it uses the launcher in a terminal. These routes always use Bash
without downloading:

- `make install-cli` or direct `scripts/install.sh` calls.
- `scripts/setup.sh` with arguments.
- Non-terminal `make install`.

Launcher failures stop setup without applying installation changes. To continue
with Bash, run `make install-cli` or `bash scripts/install.sh` with CLI options.

## Cache and recovery

Verified executables are stored in
`${XDG_CACHE_HOME:-$HOME/.cache}/agent-toolkit-tui/<version>-<target>/`.
Cache directories must be owned by the user and private (mode `700`). Downloads
are locked and published atomically; each launch checks the executable's hash.
A valid cache works offline.

Run `python3 scripts/launch_tui.py --version` to check a launch on your machine.
If the cache is corrupt, stop all launchers, remove only the affected
version/target directory, and retry.

Remove the release pin to restore Bash routing, or pin a prior compatible
release to roll back the TUI. Neither action changes the installed skills.
