# Native skill packages

Native packages copy skills, references, scripts, required helpers, and provider
adapters. They work independently of the source clone after generation. The
[repository installer](../README.md#installation-instructions) remains available:
its supporting resources link to the clone, and it owns its own settings and
provider files. Native packages never load `.env` or modify global guidance.

## Build and validate

Use Bash and Python 3.9 or newer. Generate both providers from one version:

```bash
python3 scripts/package_native.py build --version 0.1.0 --output dist/native/0.1.0
python3 scripts/package_native.py validate dist/native/0.1.0
make verify
```

Choose the real release version; `0.1.0` is an example. Output must be a new
directory. An existing output directory is preserved. Failed builds clean up
staging files and leave no partial release. ZIPs use sorted members, fixed
1980 timestamps, and normalized file modes. Identical inputs under the same
Python/zlib toolchain produce identical bytes. Generated output is ignored by Git.

Each provider directory contains:

- `plugins/<package>/`: self-contained package roots, each with the MIT license.
- `zips/<package>.zip`: archives whose root is the package root.
- A native marketplace: Claude uses `.claude-plugin/marketplace.json`; Codex uses
  `.agents/plugins/marketplace.json`.

Top-level `release.json` records the version, catalog identities, direct required
helpers, bundle membership, invocation policies, package member hashes, and ZIP
checksums. Validation checks helper closure, local Markdown links, containment,
provider metadata, member inventory, and archive bytes without extracting
untrusted archives. It needs neither the source catalog nor a provider client.
These checks detect corruption; the inventory is not a publisher signature.

Each source skill gets `agent-toolkit-<skill>`. Bundles use
`agent-toolkit-bundle-starter`, `agent-toolkit-bundle-all`, and
`agent-toolkit-bundle-<category>`. Category membership follows source directories.
The starter remains concise-output plus commit-message. A package contains each
required helper once, even when selected both directly and transitively.

## Install and invoke

For Claude Code, validate and register the generated marketplace, then install a
selection:

```bash
claude plugin validate ./dist/native/0.1.0/claude
claude plugin marketplace add ./dist/native/0.1.0/claude
claude plugin install agent-toolkit-ticket-to-plan@agent-toolkit
```

Invoke `/agent-toolkit-ticket-to-plan:ticket-to-plan`. The package also contains
plan-review at `/agent-toolkit-ticket-to-plan:plan-review`. References to bundled
helpers resolve within the same package. Required explicit invocation policies
are retained. See the [Claude manifest reference](https://code.claude.com/docs/en/plugins-reference)
and [marketplace guide](https://code.claude.com/docs/en/plugin-marketplaces).

For Codex CLI versions supporting native plugin commands:

```bash
codex plugin marketplace add ./dist/native/0.1.0/codex
codex plugin add agent-toolkit-ticket-to-plan@agent-toolkit
codex plugin list --marketplace agent-toolkit --json
```

Start a new session and select the installed skill from the skill picker, or
reference `$ticket-to-plan`. Plugin names identify packages; source skill names
stay unchanged. When several packages expose the same helper name, choose its
package in the picker and follow the package-relative helper link. Codex
archives include `.codex-plugin/plugin.json`, the skills directory declaration,
author metadata, and interface metadata. See [official OpenAI packaging guidance](https://developers.openai.com/plugins/build/plugins)
and the [ZIP submission reference](https://developers.openai.com/plugins/deploy/submission).

Standalone and bundle installations can overlap, including duplicate helpers
across packages. Installing Ruby and Rails packages together is supported at the
package level, but runtime selection of duplicate helpers still needs the
manual checks below. Prefer one route for a given skill. Native installation
does not remove managed installs, and the repository uninstaller does not manage
native plugins.

## Output, updates, and removal

Markdown workflows honor explicit user guidance configuring an output location.
Without it, they use `<repo-root>/<branch-slug>/<skill-name>/...`. A centralized
root can be specified in user guidance; packages do not embed a machine-specific
installer setting. Optional integrations absent from a package remain named
optional integrations, with the source workflow's fallback instructions intact.

For an update, generate a new version, validate it, and register or refresh its
marketplace through the provider. Retain the previous release for rollback.
Use `claude plugin update <package>@agent-toolkit` or the installed client's
plugin management UI; for a local Codex marketplace, remove its old registration with
`codex plugin marketplace remove agent-toolkit`, register the new release
location, and reinstall the selected package using `codex plugin add`. Restart the
session after changing packages. To roll back, point the provider to the retained
release and reinstall through the provider.

Remove native installations through provider management: `claude plugin uninstall
<package>@agent-toolkit`, `codex plugin remove <package>@agent-toolkit`, or the
provider's plugin UI. See the [Claude command reference](https://code.claude.com/docs/en/plugins/cli-reference)
for command and scope details. Keep generated marketplace files available while using a
local marketplace. Do not delete provider caches manually.

## Validation evidence and release preparation

Structural tests are part of `make verify`. They relocate and extract every
artifact with the source fixture unavailable, compare all resources, rebuild for
byte equality, and reject broken links, source paths, symlinks, corrupt metadata,
incomplete helper sets, modified ZIPs, and traversal members. Standard-library
unittest avoids introducing a packaging/test dependency.

Run `claude plugin validate <package-root>` for every generated Claude package
when Claude Code is installed. A runtime smoke test must separately confirm one
standalone skill, ticket-to-plan loading its packaged plan-review, and Rails
loading its packaged Ruby guidance. Also install overlapping helper packages and
confirm the intended copy is selected. Use isolated provider settings, outside
the source clone, and record the client version, commands, and observed behavior.
[Recorded evidence](native-packages-evidence.md) distinguishes installation from
actual skill invocation. Missing runtime evidence prevents a runtime-success
claim; structural validation alone does not establish model behavior.

The generated author label names project contributors; it is not a verified
publisher identity. Before public submission, choose the real publisher, provide
any required listing assets and policies, and complete provider review materials
and runtime checks. No publisher credentials, account IDs, approval status,
submission, or publication are generated by this workflow.

## Backend handoff

Source skills and `.agents/dependencies.txt` remain authoritative.
`catalog_records`, `bundle_names`, and `bundle_members` in `scripts/catalog.sh`
expose identities and requested selections; `resolve_selection` adds required
helpers. `Catalog` in `scripts/native/catalog.py` is a read-only bridge to those
same Bash operations. `Renderer` stages a selected tree; `ReleaseBuilder` adds
native manifests, marketplaces, and archives. `validate_release` works offline
from the release snapshot. `scripts/package_native.py` provides explicit build
and validate commands. Managed state and reconciliation are outside this API.
