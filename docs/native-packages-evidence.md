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
