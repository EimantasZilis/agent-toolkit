# Native package compatibility

Native packages have structural checks for both providers and recorded Codex
model-session and hosted lifecycle acceptance. Claude runtime acceptance remains pending. This is
the latest recorded evidence, dated 6 November 2026; it is not a claim about
untested client versions or public marketplace approval.

| Area | Recorded result |
| --- | --- |
| Package structure, resources, helper closure, and installer coexistence | Repository tests and `make verify` passed for both providers. |
| Codex CLI | Version `0.163.0` accepted the generated marketplace and renamed packages in an isolated home. |
| Codex model sessions | Standalone commit-message, planning with plan-review, Rails with Ruby guidance, duplicate-helper resources, and explicit-only behavior passed the recorded requests. |
| Source independence | Relocation fixtures passed with source fixtures unavailable. Model sessions selected package paths while the source clone remained on disk. |
| Claude | Manifests and resources passed structural checks. The client was unavailable; provider validation, installation, and model sessions remain unverified. |
| Hosted Codex CLI | Version `0.167.0` passed installation, skill/helper sessions, existing-install updates, removal, and recovery with hosted versions `0.1.0` and `0.1.1`; see the records below. |
| Public distribution | Hosted Git publication is recorded below. No public-directory submission or approval is claimed. |

The Codex sessions selected installed package paths explicitly. They do not
establish every picker interaction or all overlapping-helper behavior.

See the [archived test record](archive/native-packages-validation-2026-10-22.md)
for commands, fixtures, limitations, and earlier results. For the checks to run
before a release, see [release checks](native-publishing.md#release-checks).

## Git marketplace rehearsal

Date: 2026-11-06. Platform: macOS arm64. This records local generation and Git
history checks, not hosted publication or provider runtime acceptance.

Source commit: `c25d5b927d8ac70cfcb17cd2442327c2ece7c65b`. The source checkout remained on
`task/run-setup-from-source`. Snapshots `0.1.0` and `0.1.1` each contained
76 packages across both providers. The first snapshot had
695 files including its inventory document. Its `snapshot.json`
SHA-256 was `dcc53d52f2e65ad10aadafe75a4d47040b1bab95471b52cfb78362a8b03e1d12`.

Commands used the documented `snapshot --version VERSION --output PATH` and
`validate-snapshot PATH` interfaces. Two independent `0.1.0` builds had identical
inventory and release metadata; all file checksums and Git executable modes
validated. Both root catalogs resolved to their separate provider trees.
The seven automated snapshot tests also validated relocation with the fixture
source unavailable, hostile paths, missing resources, symlinks, and bad provenance.
All 19 native tests and `make verify` passed. The original archive CLI build and
validate commands passed in a new temporary directory.

A disposable source clone used an orphan `native-plugins` branch for initial
publication. A second clone from the local bare fixture received `0.1.1`, then a
new recovery commit restored the retained `0.1.0` payload:

| Operation | Local fixture commit | Result |
| --- | --- | --- |
| First version | `1bf364cf67ba003cbd8160031540812025186cc9` | `native-v0.1.0` retained |
| Second version | `dc5b57be98d8af74de3ac5d65def1310079d8ca3` | `native-v0.1.1` retained |
| Restore first payload | `e00e0481b7cd2af005c27370db6c26a0162043ee` | New forward commit; original provenance preserved |

The rehearsal checked a clean generated checkout before replacement, exact Git
index membership, both staged root catalogs, generated JSON whitespace, commit
and tag creation, validation in fresh Git checkouts, and unchanged version-tag
refs after recovery. Source Markdown retains its existing formatting; the
publication whitespace check deliberately targets generated JSON.

To honor the execution workflow’s no-push rule, the bare fixture received refs
with local `git fetch` operations. The documented hosted push commands remain a
manual maintainer step. No hosted branch, tag, provider state, or global guidance
was changed. The fixture and detailed JSON record were retained at
`/var/folders/cg/bkyxlnp114s5682077ym2wn00000gn/T/toolkit marketplace rehearsal 4cnvj4km`.

A real hosted marketplace and two reviewed published versions are still required
for Codex and Claude install/update/removal/recovery acceptance. These local
results do not establish those journeys.

## First hosted marketplace publication

Date: 2026-11-06. Platform: macOS arm64. This establishes hosted publication,
Codex marketplace registration, and package installation. Authenticated runtime,
existing-install update, removal, and recovery acceptance remain pending.

- Source commit: `2e83499bce60756814944bd05d4190398ca7fb10`, the PR #7 commit
  whose macOS and Ubuntu source-verification jobs passed.
- Published version: `0.1.0`, with 38 packages per provider and 695 snapshot files.
- Publication commit: `78968ecfb9f811e6bab6fcf512c2991464b2f88e` on
  [native-plugins](https://github.com/EimantasZilis/agent-toolkit/tree/native-plugins).
- Immutable tag: [native-v0.1.0](https://github.com/EimantasZilis/agent-toolkit/tree/native-v0.1.0).
- Retained snapshot and publication checkout:
  `/private/tmp/toolkit-hosted-acceptance.orOBgt/`.

Generation and standalone validation passed. Review checked the recorded source
and version, both catalogs, complete staged file membership, and generated JSON
whitespace. A separate orphan publication checkout supplied the commit. An atomic
push created the branch and tag together without changing source-branch history.

Codex CLI `0.167.0` used isolated `HOME` and `CODEX_HOME` directories beneath the
retained test root. The following commands succeeded:

```bash
codex plugin marketplace add EimantasZilis/agent-toolkit --ref native-plugins --json
codex plugin add commit-message@agent-toolkit --json
codex plugin add ticket-to-plan@agent-toolkit --json
codex plugin add ruby-rails-standards@agent-toolkit --json
codex plugin add ruby-standards@agent-toolkit --json
```

All four packages installed at version `0.1.0`. The fetched marketplace HEAD
matched the publication commit, and standalone snapshot validation passed against
that fetched checkout. Registration used GitHub rather than a local source or
build directory. Network access was required for fetching and publication.

The isolated Codex home reports `Not logged in`. No authentication cache was
copied and no model session has run for this hosted journey. Claude is unavailable
on PATH. A second reviewed publication and both providers' remaining lifecycle
checks are still required before tickets 6 and 7 can be marked complete.

## Hosted Codex runtime acceptance

Date: 2026-11-06. Tested surface: Codex CLI `0.167.0` on macOS arm64.
The first hosted snapshot and all four acceptance packages are version `0.1.0`,
at publication commit `78968ecfb9f811e6bab6fcf512c2991464b2f88e` and immutable
tag `native-v0.1.0`. Source commit: `2e83499bce60756814944bd05d4190398ca7fb10`.

The existing isolated environment completed `codex login --device-auth` using
the user's browser authorization. `codex login status` then reported
`Logged in using ChatGPT`. No personal authentication cache was copied.
This supersedes the unauthenticated status in the preceding section.

Retained test root: `/private/tmp/toolkit-hosted-acceptance.orOBgt/`.
`HOME` and `CODEX_HOME` point to its `user-home/` and `codex-home/` directories.
Fresh sessions used the client-selected model without a model override:

```bash
codex --no-daemon exec --strict-config --skip-git-repo-check --ephemeral \
  --ignore-rules --json -C <fixture-directory> \
  -o <test-root>/evidence/<case>-result.txt '<case prompt>'
```

The isolated permission profile extends `:workspace` and denies reads from the
source repositories, personal skill/plugin directories, retained generated
snapshots, and publication checkouts. The installed Codex executable directory
has a narrower read grant. A sandbox probe confirmed source/build reads fail
with `PermissionError` while an installed hosted skill remains readable.
Sessions use the fetched hosted marketplace and installed plugin cache.

| Case | Observed result |
| --- | --- |
| Standalone | Loaded the hosted `commit-message/0.1.0` skill and manifest, read the staged fixture README, and returned `Add README for hosted plugin acceptance`. No files were changed or committed. |
| Planning/helper | Loaded `ticket-to-plan/0.1.0` and its packaged `plan-review` helper. Produced a ticket, README index, and one phase beneath the fixture's explicit output directory. The checkpoint reported Ready to implement; the project README remained unchanged. |
| Rails/Ruby | Loaded the Rails parent, its bundled Ruby baseline, and relevant references. Identified unfiltered parameters passed to `User.create!`, including the standard strong-parameter rejection behavior. No application checks or edits ran. |
| Overlapping helpers | Loaded the Rails-relative bundled Ruby copy, then explicitly selected standalone Ruby. Their skill/core/linting hashes matched. All 26 relative links across 26 Markdown files resolved. The two selected absolute package paths remained distinct. |
| Explicit-only helper | A generic request to review a small README plan produced an assessment without a skill-loading command. The packaged `plan-review` helper was not loaded in this session. |

All five accepted sessions exited 0. JSONL traces, prompts, stderr, exit codes,
and final responses are retained beneath `evidence/`; fixtures are beneath
`runtime/`. The initial standalone attempts failed before a model session could
start because the first filesystem profile also denied the Codex executable,
then because a temporary configuration entry was placed under a trust table.
They are excluded from acceptance. The corrected run is recorded as
`hosted-standalone-0.1.0-accepted`.

These checks establish explicit package-path selection and relative helper
resolution. Picker interaction and other Codex surfaces were not tested.
An unrelated native sentinel plugin, a repository-managed sentinel skill, and
global guidance were present. A preservation baseline contains hashes for
127 files, excluding credentials, plus four unrelated/configuration sections.
All four acceptance packages were removed with `codex plugin remove
<package>@agent-toolkit --json`. The installed-package inventory became empty
and each target cache directory disappeared. All 127 preservation hashes and
four unrelated/configuration sections matched the baseline.

Recovery removed the marketplace registration and registered the immutable
hosted tag, then reinstalled the four packages:

```bash
codex plugin marketplace remove agent-toolkit --json
codex plugin marketplace add EimantasZilis/agent-toolkit --ref native-v0.1.0 --json
codex plugin add commit-message@agent-toolkit --json
codex plugin add ticket-to-plan@agent-toolkit --json
codex plugin add ruby-rails-standards@agent-toolkit --json
codex plugin add ruby-standards@agent-toolkit --json
```

The fetched HEAD matched `78968ecfb9f811e6bab6fcf512c2991464b2f88e`.
All four packages returned enabled at `0.1.0`. A fresh authenticated session
loaded the recovered standalone skill and manifest, then returned the expected
commit message; exit status was 0. All preservation hashes and sections still
matched. Before/after inventories and hash comparisons are retained beneath
`evidence/`. Personal provider configuration, unrelated native packages,
repository-managed skills, settings, and guidance were preserved.

The isolated marketplace was then registered against `native-plugins` again,
retaining the recovered `0.1.0` installations for the existing-install update
test. No source/build directory was needed during removal or recovery.

The second reviewed snapshot, `0.1.1`, was published after explicit maintainer
confirmation at publication commit
`4d4e77ebb40214900eca46b51e6442b963a52f1f`, tagged `native-v0.1.1`.
It uses the same source commit as `0.1.0`. Both catalogs, all package membership,
file modes, skill content, and resources are unchanged. Only versions and their
checksum metadata changed. Its 695-file snapshot validates, and generated JSON
whitespace checks pass. The atomic push advanced `native-plugins` and created the
new tag. A read-only remote check confirmed both refs at that commit and the
unchanged `native-v0.1.0` tag at `78968ecfb9f811e6bab6fcf512c2991464b2f88e`.

### Existing-install update and final recovery

Before refresh, all four installed cache directories contained only `0.1.0`.
No target package was removed or re-added during the following update:

```bash
codex plugin marketplace upgrade agent-toolkit --json
codex plugin list --marketplace agent-toolkit --json
```

Upgrade selected only `agent-toolkit`, reported the refreshed root, and returned
an empty errors list. Before running the list command or starting a new session,
inspection already found all four caches replaced with `0.1.1`. The fetched HEAD
matched publication commit `4d4e77ebb40214900eca46b51e6442b963a52f1f`.
The installed inventory then reported all four packages enabled at `0.1.1`.
On this client, marketplace refresh also updates configured installed packages;
there was no separate installed-package update command or reinstall.

A fresh authenticated session, `hosted-existing-update-0.1.1`, read the installed
`commit-message/0.1.1/skills/commit-message/SKILL.md` and its manifest containing
version `0.1.1`, then returned the expected staged-README commit message. It
exited 0. This establishes an observable installed-version update; skill behavior
was deliberately unchanged between these two metadata-only releases.

Codex's managed checkout contained an extra `.codex-marketplace-install.json`
receipt, so strict standalone validation of that entire directory rejected its
file inventory. Every published payload checksum still matched. A separate
pristine checkout fetched from the actual hosted `native-v0.1.1` tag passed
`validate-snapshot` without changing the validator or deleting client state.

After the update test, all four `0.1.1` plugins were removed. Their installed
inventory became empty and caches disappeared. The recovery commands above then
registered `native-v0.1.0` and restored all four enabled `0.1.0` packages. A fresh
authenticated session, `hosted-final-recovery-0.1.0`, loaded the recovered skill
and manifest, returned the expected commit message, and exited 0.

Full before/after file inventories, including additions and removals, matched
all 127 baseline hashes after final recovery. All four preserved configuration
sections also matched. The isolated fixture remains pinned to `native-v0.1.0`;
the published `native-plugins` branch remains at `0.1.1`. Neither immutable tag
was changed. Authentication was removed from the isolated home after testing.

The hosted Codex CLI lifecycle is complete for this tested surface and explicit
package selection. Claude acceptance remains pending. No packaging compatibility
change was needed. Changed-document links, Bash examples, and `git diff --check`
passed; the clean hosted snapshot validates. Structural source checks remain
covered by the CI-tested source commit because the implementation inputs did
not change.
