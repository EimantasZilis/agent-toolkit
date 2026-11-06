# Install and manage local skills

Install selected skills into Codex or Claude and save your choices for later
updates. Keep the checkout for the installer and linked resources.

## Quick start

Use a macOS or Linux terminal with Git, Bash, Make, and `shasum`. Install
Codex or Claude and sign in before continuing.

```bash
git clone https://github.com/EimantasZilis/agent-toolkit.git
cd agent-toolkit
make install
```

Terminal `make install` opens the [setup TUI](docs/usage/tui.md) from this checkout.
It requires Go at the version declared in `go.mod` (currently 1.27.3). The first
launch may download modules and a compatible toolchain. Later launches use
Go’s build cache and include source changes automatically.

Use `make install-cli` for Bash setup without Go. Choose providers and skills,
review the proposed changes, and confirm them. Fresh Bash defaults select Codex,
the starter bundle (`concise-output` and `commit-message`), repository-local
Markdown output, and no global guidance. The TUI starts with no provider selected.

Start a new agent session after setup. Run the management commands below from
the `agent-toolkit` folder.

| Option | Best for | Updates managed by |
| --- | --- | --- |
| Repository installer | Selected skills and optional global guidance; requires this clone | `make update` |
| [Native plugins](docs/usage/native-packages.md) | Skills packaged independently of the clone | Claude or Codex plugin management |

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
./scripts/install.sh --provider codex --bundle starter --full-template false --concise false --dry-run
./scripts/install.sh --provider claude --skill pr-description --full-template false --concise false
./scripts/install.sh --provider codex --bundle all --full-template true --concise true
```

Non-terminal `make install` and `scripts/setup.sh` with arguments also use Bash
without invoking Go. If source setup fails, diagnostics remain visible and
setup stops; use `make install-cli` to continue with Bash.

Repeat `--skill` for multiple skills, or use `--bundle`. CLI options override saved
settings; unspecified settings retain saved values. Fresh Bash setup defaults to
Codex and `starter`. `--list`, `--help`, and `--dry-run` write nothing. Repeat
`--provider` to select both providers in one operation:

```bash
./scripts/install.sh --provider codex --provider claude --bundle starter --dry-run
```

Explicit providers and skills replace their saved selections. Selecting fewer
can remove owned installations. Review `--dry-run` first. A terminal asks for
removal confirmation with a default of No; automation must pass
`--confirm-removals`. Decline or EOF changes nothing.

## Installed files and guidance

| Provider | Skills | Global guidance |
| --- | --- | --- |
| Codex | `${AGENTS_HOME:-~/.agents}/skills/agent-toolkit/` | `${CODEX_HOME:-~/.codex}/AGENTS.md` |
| Claude | `${CLAUDE_HOME:-~/.claude}/skills/<skill-name>/` | `${CLAUDE_HOME:-~/.claude}/CLAUDE.md` |

- Skills have a generated `SKILL.md`; supporting files link to this clone.
- Optional global guidance covers coding and Markdown output. Its writing rules
  apply to all documentation, whether created by a skill or edited directly.
  See [instructions.md](.agents/global/instructions.md).
- Existing user content is preserved. Edited skills and managed guidance blocks cause ownership conflicts.
  The whole operation stops before writes so user changes remain intact.
- Disabling guidance proposes removal of its verified managed block and requires
  confirmation. Surrounding user instructions remain intact.
- Installing `concise-output` alone does not make it a standing instruction.

## Configuration

Run `make install` to change saved providers, skills, output location, or
concise-output loading. Use `make install-cli` for Bash setup.

For scripted configuration, pass options to `scripts/install.sh`:

| Option | Effect |
| --- | --- |
| `--provider codex` or `--provider claude` | Select providers; repeat for both |
| `--skill NAME` or `--bundle NAME` | Select skills and their required helpers |
| `--output-path PATH` | Save an absolute output root; `''` selects repository output |
| `--concise true` or `false` | Enable or disable automatic concise-output loading |
| `--full-template true` or `false` | Enable or disable the full global guidance template |
| `--dry-run` | Preview without applying changes |
| `--confirm-removals` | Authorize proposed removals in automation |

Use `--concise true` for a standing instruction to load concise-output.
Disabling it keeps the skill installed. The full guidance template is a separate
choice. A shared output root may still require a minimal output-policy block.

Settings are saved in this checkout's ignored `config/` directory. Both setup
interfaces use them, and unspecified CLI options keep their saved values.
Change preferences through setup rather than editing state files.
`MARKDOWN_OUTPUT_PATH` temporarily overrides output for an invocation; an explicit
`--output-path` takes precedence and updates the saved preference.

### Markdown output

| Mode | Artifact location |
| --- | --- |
| Repository (Bash default) | `<repo-root>/<branch-slug>/<skill-name>/...` |
| Shared directory | `<output-root>/<repo-slug>/<branch-slug>/<skill-name>/...` |

Explicit paths take precedence. Slugs use lowercase names with unsafe characters
replaced by `-`; detached HEADs use `detached-<short-sha>`. Changing the output
location does not move existing artifacts. Artifacts are not committed or
removed automatically.

Skills look for earlier workflow output in this order:

1. An explicit path.
2. The current repository and branch's producer-skill directory.
3. Its configured shared-directory equivalent.
4. A documented legacy location.

## Update

Pull changes, then refresh the saved installation:

```bash
git pull --ff-only
make update
```

`make update` uses Bash without Go or selection prompts. It refreshes all saved
providers, skills, retained helpers, and output/guidance preferences. Start new
agent sessions afterward. Preview with `./scripts/update.sh --dry-run`.
Temporary `MARKDOWN_OUTPUT_PATH` overrides are ignored during update; change the
saved preference through setup instead. Provider-home overrides still select
the installation destinations.

Update requires saved settings from a successful setup. It stops
on unknown saved skills, ownership conflicts, proposed removals, or a changed
preview revision. A moved resource can leave a dangling link that ownership
records cannot safely identify; this also requires setup review. Resolve these
through `make install-cli`; preserve user changes. Apply failures roll back provider files and saved state together.

Git pull and refresh are separate operations. Linked resources can change as
soon as Git updates the clone; rollback of a failed refresh cannot undo those
source changes. Keep the checkout available and review source changes before
pulling them. A repeated refresh preserves saved preferences.

Bash and the TUI both reconcile the saved desired selection. Choosing fewer
skills or providers requires confirmation before owned paths are removed.

## Troubleshooting

| Problem | Action |
| --- | --- |
| Missing skill | Check provider and selection with `--dry-run`, then start a new session. |
| Ownership conflict or dangling resource link | Back up the affected installed skill outside the provider skills directory, then review setup. Keep the backup for comparison. |
| Broken supporting-file links | Restore the clone or reinstall from its new location. |
| Failed installation | Changes roll back; if recovery fails, the error gives the backup location. |

## Uninstall

```bash
make uninstall
```

Removes verified managed skills, guidance, and saved settings in one
transaction. Unrelated user files and native plugins remain.
Ownership conflicts stop uninstall before writes; resolve those conflicts and
retry. Failed uninstall restores provider files and saved state together.
