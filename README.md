# agent-toolkit

agent-toolkit is a small, installable set of skills for planning, delivery,
security, coding practices, communication, and career retrospectives. The
skill directories and invocation IDs are the public interface.


## Skill catalog

| Category | Pack | Purpose |
| --- | --- | --- |
| coding | annotate | Explain changed code inline without modifying files. |
| coding | define-coding-standards | Build a language standards pack. |
| coding | bash-standards | Routed Bash engineering guidance. |
| coding | python-standards | Routed Python engineering guidance. |
| coding | javascript-standards | Routed JavaScript engineering guidance. |
| coding | typescript-standards | Routed TypeScript engineering guidance. |
| coding | ruby-standards | Routed Ruby engineering guidance. |
| coding | golang-standards | Routed Go engineering guidance. |
| coding | ruby-rails-standards | Routed Ruby on Rails engineering guidance. |
| implementation | next-commit | Implement one planned unit without committing. |
| implementation | next-phase | Implement and commit a complete active phase. |
| implementation | wrap-up | Branch review and pull-request draft. |
| scoping | quick-plan | Bounded feasibility decisions. |
| scoping | plan-conventions | Shared planning and progress rules. |
| scoping | plan-review | Gate plans before implementation. |
| scoping | ticket-to-plan | Turn a ticket into a small, focused implementation plan. |
| scoping | requirements-to-tickets | Turn feature briefs into small, self-contained backlog tickets. |
| scoping | test-plan | Create a prioritized manual test plan for ticketed changes. |
| scoping | ticket-context | Build a repository-backed context brief for a ticket. |
| security | security-check | Short fix, accept, or monitor decision. |
| security | security-review | Detailed security assessment. |
| other | concise-output | Direct, complete reader-facing prose. |
| other | developer-highlights | Evidence-based career narrative from git history. |

See [CONTRIBUTING.md](CONTRIBUTING.md) for layout and validation rules.


## Installation instructions

### Prerequisites

The installer requires Bash, Make, standard Unix tools such as `awk`, `find`,
`grep`, `mktemp`, and `sed`, plus Perl's `shasum` utility.

### Configuration

```bash
cp .env.template .env
```

| Environment variable | Valid values | Controls |
| --- | --- | --- |
| `LLM_PROVIDER` | `codex`, `claude` | Selects the provider used for installation. |
| `CONCISE_OUTPUT` | `True`, `False` (default: `True`) | Enables or disables the `concise-output` skill in the LLM global config. |

### Installation
Once configured, install and verify the toolkit via:

```bash
make install
make verify
```

Installed skills are placed depending on your LLM provider.

| LLM Provider | Skills | Global config | Provider-specific metadata |
| --- | --- | --- | --- |
| Codex | `${AGENTS_HOME:-~/.agents}/skills/agent-toolkit/` | `${CODEX_HOME:-~/.codex}/AGENTS.md` | Symlinked from `.agents/providers/codex/` |
| Claude | `${CLAUDE_HOME:-~/.claude}/skills/<skill-name>/` | `${CLAUDE_HOME:-~/.claude}/CLAUDE.md` | None |

For both providers:

- Each skill contains a provider-rendered `SKILL.md`.
- Supporting files are symlinked back to `.agents/skills/`.
- The installer wraps its generated content in the provider's global
  `AGENTS.md` or `CLAUDE.md` file with `<!-- agent-toolkit:start -->` and
  `<!-- agent-toolkit:end -->` markers.
- When `CONCISE_OUTPUT=True`, it adds a nested, separately marked instruction
  block to the provider's global `AGENTS.md` or `CLAUDE.md` file.
- If the provider's global `AGENTS.md` or `CLAUDE.md` file already exists and
  is user-owned, its existing content is preserved and the marked
  agent-toolkit block is appended.
- If a previously managed global guidance file has been edited, installation
  detects the hash change, preserves the edits, and treats the file as
  user-owned on future installs.
- If a skill with the same name already exists and is user-owned, installation
  prints a warning and leaves that skill unchanged.
- If an installed skill's `SKILL.md` has been edited, its hash no longer
  matches the ownership marker; installation treats it as user-owned and
  leaves it unchanged.
- Installed skill directories contain an ownership marker, so uninstall only
  removes skills managed by this repository; same-named user skills are kept.
- `make uninstall` removes the generated `AGENTS.md` or `CLAUDE.md` content
  when it is unchanged, or removes only the marked agent-toolkit guidance from
  a user-owned file.

## Uninstall
To remove this repository's skills without affecting other global skills, run:

```bash
make uninstall
```

Note: this command checks for both claude and codex installations,
so that changing LLM providers does not leave any previous installations behind.
