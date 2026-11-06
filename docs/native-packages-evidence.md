# Native package validation evidence

Date: 2026-10-22. Branch: `task/native-packages`. Test release version: `0.1.0`.
This records local development checks, not public release or approval.

The installation commands below predate the package naming change. New builds
use skill names directly, such as `ticket-to-plan@agent-toolkit`, and bundles
such as `bundle-starter@agent-toolkit`. The earlier CLI results do not establish
runtime acceptance of the renamed packages.

## Structural evidence

- `python3 -m unittest discover -s tests -p 'test_native_*.py'`: 11 passing renderer and
  release checks cover both providers, all individual and category/starter/all
  packages, helper closure, resource relocation, explicit policies, deterministic
  rebuilds, archive extraction, corrupt artifacts, and hostile ZIP members.
- Ruff formatting, lint and complexity checks, function-size checks, and
  `git diff --check` pass for the packaging implementation.
- `python3 scripts/package_native.py build --version 0.1.0 --output
  /private/tmp/agent-toolkit-native-20261022` generated and validated 74 packages
  (37 per provider). The output is a local test artifact.
- `make verify` passed, including catalog tests, existing installer and selective
  installation suites, source validation, and isolated managed installs for both
  providers. ShellCheck is not installed; changed shell files passed `bash -n`.

## Codex installation evidence

Client: `codex-cli 0.163.0`. Working directory: `/private/tmp`.
Isolated `CODEX_HOME`:
`/private/tmp/agent-toolkit-native-smoke-20261022/codex-home`.
Every command below used that environment override:

```bash
codex plugin marketplace add /private/tmp/agent-toolkit-native-20261022/codex --json
codex plugin add agent-toolkit-commit-message@agent-toolkit --json
codex plugin add agent-toolkit-ticket-to-plan@agent-toolkit --json
codex plugin add agent-toolkit-ruby-rails-standards@agent-toolkit --json
codex plugin add agent-toolkit-ruby-standards@agent-toolkit --json
codex plugin list --marketplace agent-toolkit --json
```

All commands exited 0. The marketplace reported `agent-toolkit` with its generated
root. Each add reported version `0.1.0` and an installed path beneath the isolated
home. The final list reported all four plugins installed and enabled. The Ruby
baseline and Rails plugin coexisted, including their separate copies of Ruby
resources. No current user provider configuration was changed.

This proves CLI marketplace acceptance and installation. No model session ran
these skills, so helper invocation, implicit-invocation restrictions, and runtime
disambiguation of overlapping helpers remain unverified. Test-fixture relocation
with its source clone hidden is structural evidence, separate from these CLI
commands executed outside the repository.

## Claude and remaining runtime checks

`command -v claude` found no executable. Claude's authoritative plugin validator,
marketplace installation, and skill invocations could not run. Structural checks
cover its manifests, marketplace sources, namespaced invocation rendering,
explicit-only front matter, and required resources. They do not establish Claude
runtime success.

Before claiming runtime compatibility, run the standalone, planning/helper,
Rails/Ruby, and overlapping-helper session checks in both clients and record the
results here. No public-directory submission or approval was attempted.

## Runtime acceptance recheck

On 2026-10-22, resuming phase 1 found the implementation commits already
present. The following checks were repeated:

```bash
codex --version
command -v claude
CODEX_HOME=/private/tmp/agent-toolkit-native-smoke-20261022/codex-home codex login status
python3 scripts/package_native.py validate /private/tmp/agent-toolkit-native-20261022
```

Codex reported `codex-cli 0.163.0`. Claude was still unavailable. The isolated
Codex home reported `Not logged in` and exited 1, so no model-backed smoke
session was started. Offline release validation passed. Runtime acceptance
requires an authenticated isolated Codex home and an available Claude client
with model access; standalone, helper, explicit-only, and duplicate-helper
behavior remain unverified. No implementation commit was repeated.

The resumed `make verify` passed all 11 native tests, catalog and adapter
validation, installer and selective installation tests, and temporary managed
installation checks for both providers. `git diff --check` also passed.

## Package naming change

On 2026-10-22, package generation switched to skill names directly and
`bundle-<selection>` for bundles, retaining the `agent-toolkit` marketplace.
On branch `task/managed-reconciliation`, `make verify` passed all 12 native
tests and the repository validation and installation suites. The added test
checks unprefixed manifest identities and ZIP filenames for standalone skills
and starter/all/category bundles in both providers. Ruff formatting,
lint/complexity checks, function-size checks, and `git diff --check` passed.
Provider model-session acceptance remains pending.

## Codex runtime acceptance

On 2026-10-22, `codex-cli 0.163.0` completed model-backed native-plugin
smoke tests on branch `task/setup-tui-settings`. Claude validation and runtime
checks are deferred to the user; no Claude runtime success is claimed.

Test root: `/private/tmp/agent-toolkit-runtime-l9UPc8zi`. A fresh version
`0.1.0` release was built and validated beneath `release/`. An isolated
`codex-home/` registered that release and installed these renamed packages:

```bash
python3 scripts/package_native.py build --version 0.1.0 --output /private/tmp/agent-toolkit-runtime-l9UPc8zi/release
```

With `CODEX_HOME=/private/tmp/agent-toolkit-runtime-l9UPc8zi/codex-home`:

```bash
codex plugin marketplace add /private/tmp/agent-toolkit-runtime-l9UPc8zi/release/codex --json
codex plugin add commit-message@agent-toolkit --json
codex plugin add ticket-to-plan@agent-toolkit --json
codex plugin add ruby-rails-standards@agent-toolkit --json
codex plugin add ruby-standards@agent-toolkit --json
codex login status
```

All commands exited 0. Authentication used a protected temporary copy of the
existing ChatGPT login cache, removed after the sessions finished. User plugin
configuration and installed skills were unchanged.

Each accepted model session used this command form with the isolated home:

```bash
codex --no-daemon exec --skip-git-repo-check --ephemeral --ignore-rules \
  --sandbox <mode> --json -C <fixture-directory> \
  -o <test-root>/<case>-result.txt '<test prompt>'
```

The client selected its default model; no model override was supplied. The
planning session used `workspace-write`; the other sessions used `read-only`.
Prompts selected the native package paths explicitly and excluded managed skill
copies and the source clone. Recorded file reads stayed within the temporary
fixtures and installed packages. The source clone remained on disk; the separate
relocation tests establish behavior with source fixtures unavailable.

| Case | Request and observed result |
| --- | --- |
| `staged` | `$commit-message` read its packaged skill and the fixture's staged README. It returned `Add README for isolated native plugin smoke test` without staging or committing. |
| `planning` | `$ticket-to-plan` planned a confirmed README-only task in an empty fixture repository. It wrote the ticket, README index, and one phase, loaded its package-relative `plan-review` helper, and reported the checkpoint ready to implement. It did not implement the task or commit. |
| `rails` | `$ruby-rails-standards` reviewed an action passing untrusted parameters to `User.create!`. It loaded the bundled Ruby baseline, core/linting references, Rails application guidance, and security references, then identified the missing attribute allowlist and suggested strong parameters. |
| `helpers` | Both parent skills and their helpers loaded from package paths. All 27 local links inspected resolved. The standalone Ruby plugin and the Rails-bundled Ruby helper had identical skill, core, and linting files. Relative links selected the Rails-bundled copy; the standalone copy was inspected separately. |
| `implicit` | A generic request to review a small implementation plan used no skill-loading commands. The explicit-only `plan-review` helper was not loaded in this session. Its packaged `agents/openai.yaml` declares `allow_implicit_invocation: false`. |

All five accepted sessions exited 0. JSONL command traces, stderr, final
responses, release files, and fixture plans remain beneath the test root,
outside the source clone. These observations cover the tested requests and
explicit path selection, not every provider-picker interaction.

An earlier run with `--ignore-user-config` selected a globally installed skill;
it is excluded from native-package acceptance. A corrected run loaded the
isolated plugin configuration. Sandbox network failures required rerunning the
model sessions with network access. Some read-only commands emitted macOS
temporary-cache warnings but completed their resource checks.

The resumed `make verify` passed all 12 native tests, catalog and adapter
validation, installer and selection suites, and temporary managed-installation
verification for both providers. `git diff --check` also passed.


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
