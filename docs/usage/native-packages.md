# Native plugins

Native plugins install agent-toolkit skills into Codex or Claude Code without
keeping a source checkout. Each package contains its supporting files and
required helpers. Plugins leave the [local TUI's](../../INSTALL.md) settings
and global guidance unchanged.

## Choose a package

| Package | Includes |
| --- | --- |
| `<skill>`, such as `ticket-to-plan` | One skill and its required helpers |
| `bundle-starter` | `concise-output` and `commit-message` |
| `bundle-<category>` | All skills in a category |
| `bundle-all` | The full catalog |

Prefer one installation route per skill to avoid duplicate copies.
Markdown artifacts default to `<repo-root>/<branch-slug>/<skill-name>/...`;
an explicit path or output policy in user guidance takes precedence.

## Codex CLI

Register the hosted marketplace and install a package:

```bash
codex plugin marketplace add EimantasZilis/agent-toolkit --ref native-plugins --json
codex plugin add ticket-to-plan@agent-toolkit
codex plugin list --marketplace agent-toolkit --json
```

Start a new session and request `$ticket-to-plan`. Its `plan-review` helper is
included. If several packages contain the same helper, select the intended
package's `SKILL.md` path and follow its relative links.

To update installed packages and check their versions:

```bash
codex plugin marketplace upgrade agent-toolkit --json
codex plugin list --marketplace agent-toolkit --json
```

Start a new session after updating. Remove a package with
`codex plugin remove ticket-to-plan@agent-toolkit`.

### Restore a previous version

Replace the marketplace registration with a retained version tag and reinstall:

```bash
codex plugin remove ticket-to-plan@agent-toolkit --json
codex plugin marketplace remove agent-toolkit --json
codex plugin marketplace add EimantasZilis/agent-toolkit --ref native-v0.1.0 --json
codex plugin add ticket-to-plan@agent-toolkit --json
```

Start a new session. The selected tag applies to the whole marketplace; other
installed packages have separate caches. To follow current releases again,
replace the registration using `--ref native-plugins`.

## Claude Code

Ask the maintainer for a generated Claude release directory, then run:

```bash
claude plugin validate ./dist/native/0.1.0/claude
claude plugin marketplace add ./dist/native/0.1.0/claude
claude plugin install ticket-to-plan@agent-toolkit
```

Invoke `/ticket-to-plan:ticket-to-plan`. The included helper is
`/ticket-to-plan:plan-review`.

Register a new release and run `claude plugin update ticket-to-plan@agent-toolkit`
to update. Use `claude plugin uninstall ticket-to-plan@agent-toolkit` to remove
it. To restore an older version, register a retained release and reinstall.
Restart the session after changes.

## Local marketplaces

For a locally built Codex release, register its directory instead of the hosted
marketplace:

```bash
codex plugin marketplace add ./dist/native/0.1.0/codex
```

To update it, remove the old marketplace registration, register the new release,
and add the package again. Keep local marketplace directories available while
using their plugins. Uninstall in the local TUI removes only checkout-managed
skills; manage native plugins through the provider.

## Compatibility

Recorded checks on 6 November 2026 covered Codex CLI `0.167.0` on macOS arm64:
installation, skill and helper sessions, updates from `0.1.0` to `0.1.1`, removal,
and recovery. Sessions selected package paths explicitly; picker behavior and
other Codex surfaces were not tested. Claude packages passed structural checks;
Claude client validation and runtime acceptance remain unverified.

Maintainers: see [package builds](../maintenance/build-native.md) and
[marketplace publication](../maintenance/publish-native.md).
