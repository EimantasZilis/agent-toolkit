# Install and manage local skills

Install skills from this project into Codex or Claude. You choose which skills
to install and whether to add global guidance that applies across projects.

## Quick start

You need a macOS or Linux terminal with Git, Bash, Make, and `shasum`
available. Install Codex or Claude and sign in before continuing. 

1. Clone the repo and install it:
   
   ```bash
   git clone https://github.com/EimantasZilis/agent-toolkit.git
   make install
   ```
3. Choose Codex or Claude, then choose the skills to install. Review the
   proposed changes and confirm them. The initial defaults select the starter
   bundle (`concise-output` and `commit-message`), save generated Markdown in
   the repository where you use a skill, and leave global guidance unchanged.

4. Start a new Codex or Claude session to load the installed skills.

Run the management commands below from the `agent-toolkit` folder as well.

| Option | Best for | Updates managed by |
| --- | --- | --- |
| Repository installer | Selected skills and optional global guidance; requires keeping the project folder | `make install` |
| [Native plugins](docs/native-packages.md) | Packaged skills that do not require keeping the project folder | Claude or Codex plugin management |

The installer currently uses text prompts through Bash. When a terminal user
interface (TUI) release is configured, `make install` opens it instead; see
[TUI controls](docs/tui.md). That interface requires Python 3.9 or later and an
HTTPS connection for its first download. Afterward, a verified cached download
works offline. Go is needed only for development.

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

- Skills have a generated `SKILL.md`; supporting files link to the
  `agent-toolkit` folder you downloaded.
- Optional global guidance covers coding and Markdown output. Its writing rules
  apply to all documentation, whether created by a skill or edited directly.
  See [instructions.md](.agents/global/instructions.md).
- Existing user content is preserved. Edited skills and guidance become
  user-owned and are preserved on later installs.
- Disabling global guidance leaves any previously installed guidance in place.
- Installing `concise-output` alone does not make it a standing instruction.

## Configuration

Use the [TUI settings pages](docs/tui.md#setup-and-settings-controls) to manage
providers, skills, Markdown output, and automatic concise-output loading.
Removals require confirmation, and ownership conflicts stop the apply.
Disabling concise loading leaves the skill installed.

For scripted reconciliation, use the [managed backend reference](docs/managed-backend.md).
It owns the preview/apply commands, revision handling, selection flags, and JSON
responses. The full global template is an explicit CLI choice through
`--full-template true` or the additive installer's `--global-guidance` option.

The Bash installer saves successful choices to `.env` and reuses them next time.
Older configurations without selection or guidance settings retain `all` and
enabled guidance.

Successful installs also record shared preferences in `config/settings.env` and
actual installed skills for each provider in `config/inventory.json`. These
ignored files stay in the `agent-toolkit` folder, regardless of which folder
you run the installer from. Existing installations are imported without removing
skills. Provider ownership markers still determine which files the toolkit may
change.

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

Open a terminal in your existing `agent-toolkit` folder. Download the latest
changes and run the installer again:

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
| Broken supporting-file links | Restore the `agent-toolkit` folder to its original location, or run the installer from its new location. |
| Missing legacy output setting | Rerun interactive installation or add `MARKDOWN_OUTPUT_PATH=` to `.env`. |
| Failed installation | Changes roll back; if recovery fails, the error gives the backup location. |

## Uninstall

```bash
make uninstall
```

Removes managed skills and guidance for both providers and the installer's
settings file. User-owned skills and configuration are preserved.
