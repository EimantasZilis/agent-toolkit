# Publish a Git marketplace

A combined snapshot lets Claude and Codex read one generated marketplace branch.
Publish it to this repository’s `native-plugins` branch. Keep source development
and generated content in separate checkouts so the source remains reviewable.
The process is manual; generation does not publish anything.

The [snapshot commands](native-packages.md#combined-git-marketplace-snapshots)
validate package integrity and source provenance. They do not establish hosted
client compatibility. Keep local archive builds available for local installation.

## Generate and inspect a version

Run these blocks in the same Bash session, starting in the source checkout.
Use a new version and new output directories each time:

```bash
set -euo pipefail
TOOLKIT_SOURCE="$(git rev-parse --show-toplevel)"
TOOLKIT_VERSION=0.1.0
TOOLKIT_SOURCE_SHA="$(git -C "$TOOLKIT_SOURCE" rev-parse HEAD)"
TOOLKIT_SNAPSHOT="/tmp/agent-toolkit-native-$TOOLKIT_VERSION"
TOOLKIT_PUBLICATION="/tmp/agent-toolkit-publication-$TOOLKIT_VERSION"
TOOLKIT_REMOTE=git@github.com:EimantasZilis/agent-toolkit.git

python3 "$TOOLKIT_SOURCE/scripts/package_native.py" snapshot \
  --version "$TOOLKIT_VERSION" --output "$TOOLKIT_SNAPSHOT"
python3 "$TOOLKIT_SOURCE/scripts/package_native.py" validate-snapshot "$TOOLKIT_SNAPSHOT"
git -C "$TOOLKIT_SOURCE" show --stat "$TOOLKIT_SOURCE_SHA" -- .agents scripts LICENSE
```

Inspect `snapshot.json` and `release.json`. Confirm the source commit and version,
package membership, helper closure, and executable modes. Source inputs must
match the recorded commit. Retain each reviewed snapshot outside the source
checkout for comparison and recovery.

## Prepare a separate publication checkout

For the first publication, create an orphan branch in a new checkout. This keeps
source history out of the generated marketplace’s ancestry:

```bash
git clone --no-hardlinks --no-checkout "$TOOLKIT_SOURCE" "$TOOLKIT_PUBLICATION"
git -C "$TOOLKIT_PUBLICATION" remote set-url origin "$TOOLKIT_REMOTE"
git -C "$TOOLKIT_PUBLICATION" switch --orphan native-plugins
git -C "$TOOLKIT_PUBLICATION" fetch --tags origin
```

For later versions, use a fresh checkout of the published branch instead:

```bash
git clone --branch native-plugins "$TOOLKIT_REMOTE" "$TOOLKIT_PUBLICATION"
git -C "$TOOLKIT_PUBLICATION" fetch --tags origin
```

Use exactly one of those preparation paths. The remaining steps run against
`TOOLKIT_PUBLICATION`, the separate generated checkout.

## Replace, review, and commit the snapshot

Require a clean publication checkout on `native-plugins`. Replace only its
tracked generated content, then stage the complete new payload:

```bash
test "$(git -C "$TOOLKIT_PUBLICATION" branch --show-current)" = native-plugins
test -z "$(git -C "$TOOLKIT_PUBLICATION" status --porcelain --untracked-files=all)"
git -C "$TOOLKIT_PUBLICATION" rm -r --ignore-unmatch -- .
cp -R "$TOOLKIT_SNAPSHOT/." "$TOOLKIT_PUBLICATION/"
python3 "$TOOLKIT_SOURCE/scripts/package_native.py" validate-snapshot "$TOOLKIT_PUBLICATION"
git -C "$TOOLKIT_PUBLICATION" add --force --all -- \
  .agents .claude-plugin codex claude release.json snapshot.json
git -C "$TOOLKIT_PUBLICATION" ls-files --error-unmatch -- \
  .agents/plugins/marketplace.json .claude-plugin/marketplace.json
git -C "$TOOLKIT_PUBLICATION" diff --cached --check -- '*.json'
git -C "$TOOLKIT_PUBLICATION" diff --cached --stat
git -C "$TOOLKIT_PUBLICATION" diff --cached -- \
  .agents/plugins/marketplace.json .claude-plugin/marketplace.json snapshot.json release.json
```

The explicit `--force` on `git add` includes generated dot-directories even when
personal ignore rules hide them. It does not rewrite Git history. Review the
staged payload and both root catalogs before committing. The whitespace check
covers generated JSON. Rendered Markdown preserves the reviewed source’s existing
formatting; full-payload checksums and resource validation still cover every file.

Use an unused immutable version tag. A tag identifies the generated publication
commit; `snapshot.json` identifies its separate source commit:

```bash
if git -C "$TOOLKIT_PUBLICATION" show-ref --verify --quiet "refs/tags/native-v$TOOLKIT_VERSION"; then
  printf 'Version tag already exists; choose a new version.\n' >&2
  exit 1
fi
git -C "$TOOLKIT_PUBLICATION" commit -m "Publish native plugins $TOOLKIT_VERSION"
git -C "$TOOLKIT_PUBLICATION" tag "native-v$TOOLKIT_VERSION"
```

## Publish after review

This is the remote-write boundary. A maintainer publishes the reviewed branch
commit and new version tag together:

```bash
git -C "$TOOLKIT_PUBLICATION" push --atomic origin \
  refs/heads/native-plugins "refs/tags/native-v$TOOLKIT_VERSION"
```

Use ordinary forward commits for later versions. Existing version tags stay
unchanged. Correct a release by generating and reviewing a new version, such
as `0.1.1`, and repeating this process. A rejected push requires inspecting the
remote state and preparing a fresh publication checkout; do not force-push.

The branch advances to the current snapshot. An immutable `native-vX.Y.Z` tag
retains the exact older payload and its source provenance. Provider registration,
update, removal, and model-session acceptance must be recorded separately before
claiming a hosted installation journey works.

## Restore a retained snapshot

Recovery creates a new branch commit containing a retained, validated payload.
Use a fresh `native-plugins` checkout and point `TOOLKIT_SNAPSHOT` at that retained
snapshot. Its original version and source commit remain in `snapshot.json`:

```bash
test "$(git -C "$TOOLKIT_PUBLICATION" branch --show-current)" = native-plugins
test -z "$(git -C "$TOOLKIT_PUBLICATION" status --porcelain --untracked-files=all)"
python3 "$TOOLKIT_SOURCE/scripts/package_native.py" validate-snapshot "$TOOLKIT_SNAPSHOT"
git -C "$TOOLKIT_PUBLICATION" rm -r --ignore-unmatch -- .
cp -R "$TOOLKIT_SNAPSHOT/." "$TOOLKIT_PUBLICATION/"
git -C "$TOOLKIT_PUBLICATION" add --force --all -- \
  .agents .claude-plugin codex claude release.json snapshot.json
python3 "$TOOLKIT_SOURCE/scripts/package_native.py" validate-snapshot "$TOOLKIT_PUBLICATION"
git -C "$TOOLKIT_PUBLICATION" diff --cached --check -- '*.json'
git -C "$TOOLKIT_PUBLICATION" diff --cached --stat
git -C "$TOOLKIT_PUBLICATION" commit -m 'Restore retained native snapshot'
```

Review that recovery commit before manually publishing only the branch:

```bash
git -C "$TOOLKIT_PUBLICATION" push origin refs/heads/native-plugins
```

Keep all existing tags. Recovery changes the branch through a new commit; it
does not move the old version tag or create a second tag for that same version.
Client downgrade and recovery behavior still need provider-specific acceptance.
See the [local rehearsal evidence](native-packages-evidence.md#git-marketplace-rehearsal).
