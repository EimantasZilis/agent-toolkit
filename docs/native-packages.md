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

### Codex

For Codex CLI versions with native plugin commands:

```bash
codex plugin marketplace add ./dist/native/0.1.0/codex
codex plugin add ticket-to-plan@agent-toolkit
codex plugin list --marketplace agent-toolkit --json
```

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

See [compatibility status](native-packages-evidence.md) for tested clients and
known limits. Maintainers should follow [release checks](native-publishing.md#release-checks)
before distributing packages.
