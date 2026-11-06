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

Terminal `make install` opens the [setup TUI](docs/tui.md) from this checkout.
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
| [Native plugins](docs/native-packages.md) | Skills packaged independently of the clone | Claude or Codex plugin management |

Generate [combined Git marketplace snapshots](docs/native-packages.md#combined-git-marketplace-snapshots)
for [manual publication](docs/native-publication.md). Hosted client acceptance
is separate from local package validation.

The [hosted Codex CLI journey](docs/native-packages.md#codex-cli) is verified on
CLI `0.167.0` for installation, skill/helper sessions, existing-install updates,
removal, and recovery. See the [evidence](docs/native-packages-evidence.md#hosted-codex-runtime-acceptance)
for the tested versions and limitations. Claude client acceptance remains pending.

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
global template remains an explicit CLI choice through `--full-template true`.

Bash setup and the TUI use one managed planner and transaction. Successful
applies save desired settings in `config/settings.env` and the observed inventory
in `config/inventory.json`. Both ignored files belong to this clone, independent
of the caller’s directory. Provider files and saved state roll back together.

Setup reads only this canonical state. Without saved state, Bash uses its fresh
defaults and the TUI starts with no provider selected. Use setup options or the
TUI to change saved choices.

| Setting | Values | Purpose |
| --- | --- | --- |
| `PROVIDERS` | Space-separated `codex` and/or `claude` | Desired providers |
| `REQUESTED_SKILLS` | Space-separated skill names | Desired skills; bundles expand when selected |
| `RETAINED_HELPERS` | Skill names | Installed helpers retained after parent removal |
| `GLOBAL_GUIDANCE` | `true`, `false` | Full global template |
| `CONCISE_OUTPUT` | `true`, `false` | Concise loading independently of the full template |
| `MARKDOWN_OUTPUT_PATH` | Blank or absolute path | Repository or central Markdown storage |

Use `--output-path PATH` to save an output root (`''` selects repo-local).
`MARKDOWN_OUTPUT_PATH` remains a temporary rendering override; it does not
replace the saved preference. An explicit `--output-path` takes precedence.
Use `--concise` and `--full-template` to control these settings independently.
An output root can still require a minimal output-policy block.

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

Update requires canonical state from a successful setup. It stops
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

Removes verified managed skills, guidance, and canonical settings in one
transaction. Unrelated user files and native plugins remain.
Ownership conflicts stop uninstall before writes; resolve those conflicts and
retry. Failed uninstall restores provider files and saved state together.
