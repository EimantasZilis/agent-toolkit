# Build native packages

Build self-contained Codex and Claude plugins from the shared skill library.
Use a Git snapshot for hosted publication or an archive release for local
marketplaces and ZIP distribution. Both require Bash and Python 3.9 or newer.

## Git marketplace snapshot

Commit source inputs before generating, then choose a new version and output
directory:

```bash
python3 scripts/package_native.py snapshot --version 0.1.0 --output /tmp/native-snapshot-0.1.0
python3 scripts/package_native.py validate-snapshot /tmp/native-snapshot-0.1.0
```

The snapshot contains both provider catalogs and separate Codex and Claude
package trees. `snapshot.json` records the source commit, version, file checksums,
and executable modes. `release.json` lists package contents and invocation rules.
Follow the [publication guide](publish-native.md) to publish the reviewed tree.

Source inputs in `.agents/`, `scripts/`, and `LICENSE` must match HEAD, including
executable modes. Untracked inputs and ignored skill resources are rejected;
Python bytecode under `scripts/**/__pycache__/` is excluded. Unrelated documentation
and managed settings need not be committed.

Validate a pristine snapshot or clean checkout of its published tag. Codex adds
an installation receipt to its managed checkout, which fails the snapshot's
strict file inventory check.

## Archive release

```bash
python3 scripts/package_native.py build --version 0.1.0 --output dist/native/0.1.0
python3 scripts/package_native.py validate dist/native/0.1.0
```

Each provider directory contains a marketplace catalog, self-contained packages
under `plugins/`, and archives under `zips/`. The top-level `release.json` records
package contents and checksums. Users can [install these plugins](../usage/native-packages.md)
without keeping the source checkout.

## Release checks

1. Run `make verify` for package and installer checks.
2. Validate the generated snapshot or archive release. Validation checks file
   integrity, required helpers, local links, and package boundaries without
   needing a provider client or the source tree.
3. Run `claude plugin validate <package-root>` for each Claude plugin.
4. Test a standalone skill, ticket-to-plan with plan-review, Rails with Ruby
   guidance, and selection when packages contain duplicate helpers.
5. Test installation, existing-install updates, removal, and recovery in the
   clients you intend to support. Record client versions and results outside
   the source clone.

Checksums establish consistency, not publisher authenticity or runtime behavior.
See [compatibility](../usage/native-packages.md#compatibility) for the tested clients.
Public-directory submission requires separate publisher details and provider review.
