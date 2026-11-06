# Native skill packages

Native packages install agent-toolkit skills as Claude or Codex plugins.
They include the skills, supporting files, and required helpers, so the source
clone is no longer needed after building.

The [repository installer](../INSTALL.md) is an
alternative that keeps supporting files linked to the clone. Native packages
leave its settings and global guidance unchanged.

## Available packages

| Package | Includes |
| --- | --- |
| `<skill>` | One skill and its required helpers |
| `bundle-starter` | `concise-output` and `commit-message` |
| `bundle-<category>` | All skills in a source category directory |
| `bundle-all` | The full catalog |

Each skill's plugin uses the skill name, such as `ticket-to-plan`.
Each package includes a required helper only once, even if several skills need it.

Markdown output defaults to `<repo-root>/<branch-slug>/<skill-name>/...`.
An explicit path or output policy in user guidance takes precedence.

For a locally generated marketplace, ask its maintainer for the provider's
release directory. Maintainers can [build and validate a release](native-publishing.md).

## Combined Git marketplace snapshots

Generate a new self-contained Git tree with both provider catalogs at its root:

```bash
python3 scripts/package_native.py snapshot --version 0.1.0 --output /tmp/native-snapshot-0.1.0
python3 scripts/package_native.py validate-snapshot /tmp/native-snapshot-0.1.0
```

The snapshot has Codex `.agents/plugins/marketplace.json` and Claude
`.claude-plugin/marketplace.json`. Their sources resolve to distinct
`codex/plugins/<package>` and `claude/plugins/<package>` trees. ZIP archives
remain available through `build`; the Git snapshot contains package trees.

Commit the source inputs before generating. `.agents/`, `scripts/`, and `LICENSE`
must match HEAD, including executable bits. Untracked inputs and ignored skill
resources are rejected. Python bytecode in `__pycache__` directories under `scripts/` is excluded
from source-input checks. Unrelated documentation or managed settings do not
change the generated payload and need not be committed for generation.

`snapshot.json` records the exact source commit, package version, every payload
file checksum, and Git executable modes. `release.json` preserves catalog,
bundle, helper, resource, and invocation-policy contracts. Validation rejects
symlinks, escaping or cross-provider catalog paths, broken resources, missing
licenses, changed bytes, and changed executable modes. Only the root `.git`
administration entry is ignored, allowing validation in a publication checkout.
The snapshot can be relocated and validated without its source tree.

Validate a pristine snapshot or clean Git checkout. Codex adds its own
`.codex-marketplace-install.json` receipt to managed marketplace checkouts;
that extra client file is outside the published inventory and fails the strict
standalone check. Validate a separate clean checkout of the published tag instead.

Checksums prove internal consistency, not publisher authenticity. Review the
recorded source commit and the generated tree before any manual publication.
Hosted registration and update acceptance require separate real client checks.
Use the [manual publication process](native-publication.md) to review and publish
the generated tree to `native-plugins` with immutable version tags.

## Install and invoke

Use `plugin-name@marketplace-name` to select a package: for example,
`ticket-to-plan@agent-toolkit` or `bundle-starter@agent-toolkit`.

### Claude Code

Validate and register the generated marketplace, then install a package:

```bash
claude plugin validate ./dist/native/0.1.0/claude
claude plugin marketplace add ./dist/native/0.1.0/claude
claude plugin install ticket-to-plan@agent-toolkit
```

Invoke `/ticket-to-plan:ticket-to-plan`. Its required helper is
included as `/ticket-to-plan:plan-review`.

### Codex CLI

Install from the hosted Git marketplace without cloning or building the toolkit:

```bash
codex plugin marketplace add EimantasZilis/agent-toolkit --ref native-plugins --json
codex plugin add ticket-to-plan@agent-toolkit
codex plugin list --marketplace agent-toolkit --json
```

The Git marketplace contains both providers' packages. Codex selects its own
catalog. `native-plugins` tracks the current published snapshot; an immutable
tag such as `native-v0.1.0` selects a retained version instead.

Start a new session after installing. Request `$ticket-to-plan`; its required
`plan-review` helper is included in the package. When multiple packages contain
the same helper, explicitly select the intended package's `SKILL.md` path and
follow its relative resource links. For example, the Rails package links to
its own bundled Ruby baseline even when standalone Ruby is also installed.

The hosted runtime checks use Codex CLI `0.167.0` on macOS arm64 and explicit
package-path selection. They do not establish picker behavior or acceptance
on other Codex surfaces. See the [recorded evidence](native-packages-evidence.md).
Prefer one installation route per skill to avoid duplicates between native
plugins, bundles, and repository-managed copies.

For a locally built archive release, register its Codex directory instead:

```bash
codex plugin marketplace add ./dist/native/0.1.0/codex
```

Keep that local marketplace directory available while using its plugins.

## Update or remove

| Action | Claude Code | Codex |
| --- | --- | --- |
| Update | Register the new release and run `claude plugin update <package>@agent-toolkit` | For hosted Git, run `codex plugin marketplace upgrade agent-toolkit --json` |
| Remove | `claude plugin uninstall <package>@agent-toolkit` | `codex plugin remove <package>@agent-toolkit` |
| Roll back | Register a retained release and reinstall | Register a retained release and reinstall |

Validate each new release and restart the session after changing packages.
The repository's `make uninstall` does not remove native plugins.

On the tested Codex CLI `0.167.0`, marketplace upgrade refreshed both the Git
catalog and all four existing installed package caches from `0.1.0` to `0.1.1`.
No separate `plugin add` or reinstall was needed. Inspect the installed versions
and start a new session to use the updated files:

```bash
codex plugin marketplace upgrade agent-toolkit --json
codex plugin list --marketplace agent-toolkit --json
```

For a locally built Codex release, remove the old marketplace registration,
register the new local release directory, and add the package again. Keep that
local marketplace directory available while using its plugins.

### Recover a hosted Codex version

Remove the affected package and replace the marketplace registration with a
retained immutable version. For example:

```bash
codex plugin remove ticket-to-plan@agent-toolkit --json
codex plugin marketplace remove agent-toolkit --json
codex plugin marketplace add EimantasZilis/agent-toolkit --ref native-v0.1.0 --json
codex plugin add ticket-to-plan@agent-toolkit --json
```

Start a new session and check the recovered package. The selected ref applies
to the whole marketplace; other installed packages have their own caches.
To track current releases again, replace that registration using the same
remove/add commands with `--ref native-plugins`.

See [compatibility status](native-packages-evidence.md) for tested clients and
known limits. Maintainers should follow [release checks](native-publishing.md#release-checks)
before distributing packages.
