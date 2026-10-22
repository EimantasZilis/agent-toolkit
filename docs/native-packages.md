# Native skill packages

Native packages install agent-toolkit skills as Claude or Codex plugins.
They include the skills, supporting files, and required helpers, so the source
clone is no longer needed after building.

The [repository installer](../README.md#installation-instructions) is an
alternative that keeps supporting files linked to the clone. Native packages
leave its settings and global guidance unchanged.

## Available packages

| Package | Includes |
| --- | --- |
| `agent-toolkit-<skill>` | One skill and its required helpers |
| `agent-toolkit-bundle-starter` | `concise-output` and `commit-message` |
| `agent-toolkit-bundle-<category>` | All skills in a catalog category |
| `agent-toolkit-bundle-all` | The full catalog |

Markdown output defaults to `<repo-root>/<branch-slug>/<skill-name>/...`.
An explicit path or output policy in user guidance takes precedence.

## Build

Requires Bash and Python 3.9 or newer. Replace `0.1.0` with your release version
and choose a new output directory:

```bash
python3 scripts/package_native.py build --version 0.1.0 --output dist/native/0.1.0
python3 scripts/package_native.py validate dist/native/0.1.0
```

The release contains Claude and Codex marketplaces, plugin folders, ZIP archives,
and a `release.json` inventory with checksums. Validation checks package contents,
required helpers, links, metadata, and archives. It works without the source
clone or a provider client.

## Install and use

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

Each source skill uses its own name as the plugin identifier, such as
`ticket-to-plan`. Bundles use `bundle-starter`, `bundle-all`, and
`bundle-<category>`. Category membership follows source directories.
The starter remains concise-output plus commit-message. A package contains each
required helper once, even when selected both directly and transitively.

## Install and invoke

For Claude Code, validate and register the generated marketplace, then install a
selection:

```bash
claude plugin validate ./dist/native/0.1.0/claude
claude plugin marketplace add ./dist/native/0.1.0/claude
claude plugin install ticket-to-plan@agent-toolkit
```

Invoke `/ticket-to-plan:ticket-to-plan`. Its required helper is
included as `/ticket-to-plan:plan-review`.

### Codex

For Codex CLI versions with native plugin commands:

```bash
codex plugin marketplace add ./dist/native/0.1.0/codex
codex plugin add ticket-to-plan@agent-toolkit
codex plugin list --marketplace agent-toolkit --json
```

Both clients use `plugin-name@marketplace-name` for these installation commands:
`ticket-to-plan` is the plugin, and `agent-toolkit` is the marketplace. For
example, install the starter bundle as `bundle-starter@agent-toolkit`.

Start a new session and choose the skill in the picker, or use `$ticket-to-plan`.
If multiple packages contain the same helper, choose the intended package in
the picker. Prefer one installation route per skill to avoid duplicates.

## Update or remove

Keep local marketplace files available while using the plugins.

| Action | Claude Code | Codex |
| --- | --- | --- |
| Update | Register the new release and run `claude plugin update <package>@agent-toolkit` | Remove the old marketplace registration with `codex plugin marketplace remove agent-toolkit`, register the new release, and add the package again |
| Remove | `claude plugin uninstall <package>@agent-toolkit` | `codex plugin remove <package>@agent-toolkit` |
| Roll back | Register a retained release and reinstall | Register a retained release and reinstall |

Validate each new release and restart the session after changing packages.
The repository's `make uninstall` does not remove native plugins.

## Release checks

- Run `make verify` for package and installer checks.
- Validate each generated Claude plugin with `claude plugin validate <package-root>`.
- Test a standalone skill, ticket-to-plan with plan-review, Rails with Ruby
  guidance, and selection when packages contain duplicate helpers.
- Record client versions and observed results outside the source clone.
  [Recorded evidence](native-packages-evidence.md) shows what has been checked.

Package validation checks file integrity; it does not prove runtime behavior
or verify the publisher. Public submission still requires publisher details,
listing materials, and provider review.

Provider references: [Claude plugins](https://code.claude.com/docs/en/plugins-reference),
[Claude marketplaces](https://code.claude.com/docs/en/plugin-marketplaces),
[Claude commands](https://code.claude.com/docs/en/plugins/cli-reference),
[Codex packaging](https://developers.openai.com/plugins/build/plugins), and
[ZIP submissions](https://developers.openai.com/plugins/deploy/submission).
