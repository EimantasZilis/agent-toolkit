# Install and manage local skills

Install from this checkout to choose skills and optional global guidance for
Codex or Claude. Supporting files remain linked to the checkout, so keep it
available after installation.

## Quick start

Requires Git, Bash, Make, `shasum`, and an installed, signed-in provider client.
Run these commands in a macOS or Linux terminal:

```bash
git clone https://github.com/EimantasZilis/agent-toolkit.git
cd agent-toolkit
make install
```

This checkout uses Bash until a TUI release is pinned. Choose a provider and
skills when prompted, review the changes, then start a new client session.
Fresh defaults select the starter bundle, repository-local Markdown output,
and no changes to global guidance. See [TUI controls](docs/tui.md) for the
terminal interface when enabled.

| Option | Best for | Updates managed by |
| --- | --- | --- |
| Repository installer | Selected skills and optional global guidance; requires this clone | `make install` |
| [Native plugins](docs/native-packages.md) | Skills packaged independently of the clone | Claude or Codex plugin management |

## Select skills

Run `make install-cli` and choose a bundle or space-separated skill names:

| Selection | Includes |
| --- | --- |
| `starter` | `concise-output` and `commit-message` |
| `all` | Full catalog |
| Category, such as `coding` | All skills in that category |
| Skill names | Only those skills and their required helpers |

Required helpers are included automatically: `plan-review` for `ticket-to-plan`
and `ruby-standards` for `ruby-rails-standards`. Optional skills are separate.

For non-interactive installation:

```bash
./scripts/install.sh --list
./scripts/install.sh --provider codex --bundle starter --no-global-guidance --dry-run
./scripts/install.sh --provider claude --skill pr-description --no-global-guidance
./scripts/install.sh --provider codex --bundle all --global-guidance
```

Non-terminal `make install` and `scripts/setup.sh` with arguments also use Bash
without downloading. If the TUI launcher fails, setup stops; use
`make install-cli` to continue with Bash. See
[launcher and cache details](docs/tui-releases.md#verified-launcher) and
[release dispatch and pin review](docs/tui-releases.md#dispatch-and-review).

Repeat `--skill` for multiple skills, or use `--bundle`. CLI options override saved
settings; unspecified settings use saved values or defaults. `--list` and
`--dry-run` write nothing. Install separately for each provider.

## Installed files and guidance

| Provider | Skills | Global guidance |
| --- | --- | --- |
| Codex | `${AGENTS_HOME:-~/.agents}/skills/agent-toolkit/` | `${CODEX_HOME:-~/.codex}/AGENTS.md` |
| Claude | `${CLAUDE_HOME:-~/.claude}/skills/<skill-name>/` | `${CLAUDE_HOME:-~/.claude}/CLAUDE.md` |

- Skills have a generated `SKILL.md`; supporting files link to this clone.
- Optional global guidance covers coding and Markdown output. Its writing rules
  apply to all documentation, whether created by a skill or edited directly.
  See [instructions.md](.agents/global/instructions.md).
- Existing user content is preserved. Edited skills and guidance become
  user-owned and are preserved on later installs.
- Disabling global guidance leaves any previously installed guidance in place.
- Installing `concise-output` alone does not make it a standing instruction.

## Configuration

For explicit reconciliation across providers, use `scripts/managed.sh preview`
with repeated `--provider` and `--skill` arguments. Apply the same selection with
`scripts/managed.sh apply --revision REVISION`, using the revision from preview.
Removals require `--confirm-removals`; conflicts stop the whole apply.
`--no-providers` removes owned installations, and `--remove-skill NAME` removes
an unused skill or helper. Required helpers cannot be removed while selected
skills need them. Native plugins and user-owned files are preserved.
The [versioned backend contract](docs/managed-backend.md) describes its JSON
responses, read-only status/catalog commands, revisions, and errors.

Managed reconciliation accepts `--output-path PATH` (blank for repo-local output)
and `--concise true|false` for shared output and automatic-loading preferences.
The [setup and settings TUI](docs/tui.md) exposes these controls with explicit
page saves, final setup review, and confirmations for removals and directory creation.
Disabling concise loading keeps the skill installed. These controls update only
the toolkit's guidance block and preserve surrounding instructions. The full
global template remains an explicit CLI choice through `--full-template true`
or the legacy installer's `--global-guidance` option.

The Bash installer saves successful choices to `.env` and reuses them next time.
Older configurations without selection or guidance settings retain `all` and
enabled guidance.

Successful installs also record shared preferences in `config/settings.env` and
actual installed skills for each provider in `config/inventory.json`. These
ignored files stay under this toolkit clone, regardless of the caller's working
directory. Existing installations are imported without removing skills; provider
ownership markers still determine which files the toolkit may change.

| Setting | Values | Purpose |
| --- | --- | --- |
| `LLM_PROVIDER` | `codex`, `claude` | Installation provider |
| `SKILL_SELECTION` | Bundle or space-separated skill names | Requested skills |
| `GLOBAL_GUIDANCE` | `True`, `False` | Update global guidance |
| `MARKDOWN_OUTPUT_PATH` | Blank or absolute path | Repository or central Markdown storage |
| `CONCISE_OUTPUT` | `True`, `False` | Standing concise output with global guidance; includes the skill |

Use `AGENT_TOOLKIT_ENV_FILE` for a different settings file, including separate
provider selections. `MARKDOWN_OUTPUT_PATH` also accepts a temporary environment
override without changing the saved setting.

| Output mode | Artifact location |
| --- | --- |
| Repository (default) | `<repo-root>/<branch-slug>/<skill-name>/...` |
| Central | `<output-root>/<repo-slug>/<branch-slug>/<skill-name>/...` |

Explicit paths take precedence. Slugs use lowercase names with unsafe characters
replaced by `-`; detached HEADs use `detached-<short-sha>`. Artifacts are not
committed or removed automatically.

Skills look for earlier workflow output in this order:

1. An explicit path.
2. The current repository and branch's producer-skill directory.
3. Its configured central equivalent.
4. A documented legacy location.

## Update

Pull changes, then run setup again:

```bash
git pull --ff-only
make install
```

For saved settings without prompts, use `./scripts/install.sh --provider codex`
(or `claude`). Restart the client session afterward.

The Bash installer is additive: choosing fewer skills keeps earlier installs.
The managed TUI supports removals with confirmation. The former
`plan-conventions` skill also remains installed; its replacements are
`plan-writing-conventions` and `plan-execution-conventions`.

## Troubleshooting

| Problem | Action |
| --- | --- |
| Missing skill | Check provider and selection with `--dry-run`, then start a new session. |
| User-owned skill skipped | Keep your version or move it aside before reinstalling. |
| Broken supporting-file links | Restore the clone or reinstall from its new location. |
| Missing legacy output setting | Rerun interactive installation or add `MARKDOWN_OUTPUT_PATH=` to `.env`. |
| Failed installation | Changes roll back; if recovery fails, the error gives the backup location. |

## Uninstall

```bash
make uninstall
```

Removes managed skills and guidance for both providers and the installer's
settings file. User-owned skills and configuration are preserved.
