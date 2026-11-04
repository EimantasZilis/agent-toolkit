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

3. Validate the complete release using the command below. On each available
   target, run `agent-toolkit --version` and try the UI. Record which targets you ran.
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

## Validate release files

Use the read-only validator to check all four archives, `SHA256SUMS`, and
`release.json` together:

```sh
python3 scripts/validate_tui_release.py --version v0.1.0 --directory dist/tui/v0.1.0
```

The command checks archive and executable hashes, exact embedded metadata,
member paths and permissions, and the complete target and checksum lists.
It rejects duplicate JSON fields and ZIP members. Input files and ZIP members
are limited to 64 MiB each. ZIP members must use the builder's DEFLATE
compression; other methods are rejected before decompression. The validator
never extracts or executes binaries, builds, downloads, or changes the supplied
files. Unrelated files in the directory are ignored. A validation failure
names the affected file or field and exits with a nonzero status.

For a downloaded release, retain the original build's `release.json` separately
and compare against those bytes:

```sh
python3 scripts/validate_tui_release.py --version v0.1.0 \
  --directory /tmp/downloaded-release \
  --expected-manifest /path/to/retained-build/release.json
```

Without `--expected-manifest`, success establishes internal consistency only.
Someone who replaces the archives and both hash records can produce another
internally consistent release. The optional argument requires an exact byte
match with the separately retained manifest, then checks the archives against
its hashes. Use trusted build evidence for that file; a manifest downloaded
alongside the release provides no independent comparison. This command does
not establish publication, provenance, runtime behavior, or visual approval.

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

## CI candidates

By default, the **TUI release candidate** Actions workflow builds and retains a
checked candidate without publishing a release or changing the launcher pin.
Dispatch it from the repository's default branch after this workflow is merged
there.
Supply a version such as `v0.1.0`. Leave `source_ref` empty to use the dispatch
SHA, or supply a commit/ref reachable from the default branch. Preflight rejects
invalid versions and untrusted sources before Go setup or build output.

The workflow requires repository Actions access to these native runners:

| Target | Runner |
| --- | --- |
| linux-amd64 | ubuntu-24.04 |
| linux-arm64 | ubuntu-24.04-arm |
| darwin-amd64 | macos-15-intel |
| darwin-arm64 | macos-15 |

Runner availability and repository Actions permissions must be confirmed by a
hosted run. A queued, missing, cancelled, or failed runner cannot supply passing
runtime evidence. Runtime jobs download the original payload by artifact ID;
they do not rebuild it. Each checks the native host, exact version output,
loaded screen, clean quit, and absence of managed writes. PTY failures are
failures. Only an explicitly documented unsupported capability can be recorded
as a PTY skip; the current harness does not automatically skip any checks.
Native version checks always have to pass. Visual review remains manual.

Open the run's **aggregate** summary and retain its artifact IDs. Download
`tui-result-<attempt>` first. Only `candidate-result.json` with `status` set to
`approved` is usable by the later publication stage. It contains the source and
workflow SHAs, original payload run/attempt and ID, six SHA-256 digests, report
artifact IDs, current verification attempt, required-job results, per-target
outcomes, and explicit PTY skips. Failure artifacts are diagnostics only.

Retrieve the payload using that exact `payload_id`, rather than choosing an
artifact named “latest”. The Actions REST endpoint is
`GET /repos/{owner}/{repo}/actions/artifacts/{artifact_id}/zip`. Extract the
six files into an empty directory and independently run the release validator.
Keep `candidate-result.json` separately and compare all six file digests with
its `hashes` before any later publication. The evidence is not a seventh release
asset or a launcher manifest extension.

All artifacts expire after 30 days. An expired candidate requires a new run.
A partial retry preserves the original payload identity, but approval requires
all four reports from the current verification attempt. If a partial retry
cannot supply all reports, rerun all jobs to create and check a new candidate;
do not substitute old reports or rebuild under an existing payload ID.
Per-version concurrency prevents overlapping candidate runs. With `publish`
left false, no release tags or assets are written. Source pins and activation
PRs are never changed by the candidate or publication stages.

For a local native runtime check after building all four targets:

```sh
python3 scripts/smoke_tui_release.py --version v0.1.0 \
  --target darwin-arm64 --directory dist/tui/v0.1.0 \
  --checkout . --report /tmp/tui-runtime-report.json
```

Choose the target matching the actual host. This verifies only that target;
it does not establish hosted runner availability or manual visual approval.

## Publish a checked candidate

Set the workflow's `publish` option to true to publish the candidate produced
by that same run. It defaults to false. Publication does not rebuild binaries,
change the launcher pin, or provide manual visual approval.

A read-only job checks the approved aggregate, original native reports, source
and workflow ancestry, and all seven retained evidence/payload artifact IDs.
Every artifact must belong to that workflow run and remain available. The
publication job repeats these checks before its first release write. It runs
only trusted workflow tooling; it never executes release binaries or backend
scripts from the release source. Only this job receives `contents: write`;
artifact checks also need `actions: read`. Other jobs keep read-only access.

The publication sequence is:

1. Validate the retained six files and their candidate hashes. Require the
   requested repository, version, source SHA, original run/attempt, payload ID,
   report IDs, and verification attempt to match trusted workflow outputs.
2. Require an existing lightweight or annotated tag to resolve to the checked
   commit. Create a missing tag at that commit. Never move an existing tag.
3. Create an unverified draft whose body records the exact candidate identity.
   Upload only the four archives, `SHA256SUMS`, and `release.json`.
4. Download all six draft assets through the authenticated API. Compare each
   with retained hashes and validate against the retained manifest.
5. Expose the release while keeping its unverified warning. Download all six
   public URLs used by the launcher without credentials, then repeat hash and
   manifest validation. Availability errors get at most three attempts per
   file; a checksum mismatch fails immediately.
6. Remove the warning only after public verification succeeds. Retain
   `publication-report.json` in `tui-publication-<attempt>` for 30 days.

Only a report with `status: verified`, passed draft/public checks, matching
candidate identity, and six matching expected/actual hashes can be used by the
later activation stage. The workflow's publication job must also succeed.
The report includes release ID/tag/URL, completed steps, original candidate
identity, current publication attempt, and failures. An unverified report or
failed job blocks activation even if a tag or public release already exists.

Existing releases are conflicts unless their ownership marker identifies the
same candidate and their complete asset set matches retained bytes. Identical
public retries reverify without uploads. A complete owned draft can finish
publication after fresh checks. A partial upload, duplicate name, extra asset,
different candidate, changed tag, or changed download fails without replacing
assets or moving tags. A new run creates a new candidate and cannot silently
adopt an old release, even when both candidates use the same source commit.

The implementation follows GitHub's [release API](https://docs.github.com/en/rest/releases/releases)
and [asset download API](https://docs.github.com/en/rest/releases/assets).
Authenticated asset redirects strip credentials before contacting storage.
All downloads require HTTPS and are bounded to 64 MiB each.

## Recover an unverified publication

Inspect the failed job summary and retained publication report first. Record
the release ID, tag, candidate run/payload IDs, completed steps, and failed asset.
A warning update can itself fail; an unverified report still blocks activation.

- For a transient download failure with a complete owned draft/public release,
  retry only publication while the original candidate and all evidence remain
  retained. Original run/attempt and payload identity stay unchanged.
- For a partial upload, stop and inspect ownership and tag/SHA agreement. A
  maintainer may remove only that failed attempt's owned draft and its assets,
  then retry publication of the original retained candidate. Leave the tag at
  its checked commit. The helper does not delete releases/assets or repair
  partial uploads automatically.
- For a conflicting or unverified public release, do not overwrite assets or
  move the tag. Resolve it manually or choose a new version and candidate.
- If artifacts have expired, start a new candidate run with a new version.

Before relying on publication, test a unique version in a disposable repository
with trusted workflow code. Verify downloaded public bytes, the final tag SHA,
identical-candidate retries, and the retained report. Local fake-API tests do
not establish live publication or hosted runner availability.
