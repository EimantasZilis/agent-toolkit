# Build and distribute native packages

Build self-contained Codex and Claude packages from the shared skills, then
validate them before distribution. Users can [install and manage the resulting
plugins](native-packages.md) without keeping the source checkout.

## Build

Requires Bash and Python 3.9 or newer. Replace `0.1.0` with your release version
and choose a new output directory:

```bash
python3 scripts/package_native.py build --version 0.1.0 --output dist/native/0.1.0
python3 scripts/package_native.py validate dist/native/0.1.0
```

Each provider's release directory contains:

- `plugins/<package>/`: self-contained package roots, each with the MIT license.
- `zips/<package>.zip`: archives whose root is the package root.
- A native marketplace: Claude uses `.claude-plugin/marketplace.json`; Codex uses
  `.agents/plugins/marketplace.json`.

The release's top-level `release.json` records the version, skills, required
helpers, bundle contents, invocation rules, and file and ZIP checksums.
Validation checks that required helpers are included, local Markdown links
resolve, paths stay within the package, and metadata, files, and archives match
the inventory. It checks ZIP files without extracting them and needs neither
the source clone nor a provider client.

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
