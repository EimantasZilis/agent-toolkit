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
