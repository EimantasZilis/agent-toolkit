# agent-toolkit

agent-toolkit gives Codex and Claude reusable skills for everyday
development tasks such concise communication, coding
standards, turning a feature request into implementation plan and code review.

## Start with two skills

The starter selection includes:

- **`commit-message`:** draft a short message describing your staged changes.
- **`concise-output`:** ask for clear answers with useful detail and less repetition.

Try these before exploring the full catalog. Each skill works on its own;
you do not need to follow a planning workflow.

## Install the starter selection

The local terminal UI lets you install the starter skills for Codex, Claude
Code, or both. It also lets you configure guidance and Markdown output.

**Prerequisites:** Git, the Go version declared in `go.mod`, and an installed,
signed-in Codex or Claude Code client. Make is optional. Run the following in a
macOS or Linux terminal:

```bash
git clone https://github.com/EimantasZilis/agent-toolkit.git
cd agent-toolkit
make run
```

`make run` opens the [local management TUI](docs/usage/tui.md) from current
source. Without Make, use `go run ./cmd/agent-toolkit`. The first launch may
download modules and a compatible toolchain.

1. Select Codex, Claude Code, or both. No provider is selected initially.
2. Keep the preselected starter skills and work through the settings pages.
3. Review the proposed paths and apply the installation.
4. Start a new agent session in the repository where you want to use the skills.

Keep this clone: supporting files link back to it. For controls and settings,
see the [TUI guide](docs/usage/tui.md) and [installation guide](INSTALL.md).

Prefer provider-managed plugins without a source checkout? See the
[native plugin guide](docs/usage/native-packages.md) for installation and recorded
compatibility. Codex CLI runtime checks are recorded there; Claude client runtime
acceptance remains unverified. Choose one installation route per skill.

## Try your first task

In a repository with changes you have already reviewed and staged, ask your
agent to draft a commit message:

| Local installation | Prompt |
| --- | --- |
| Codex | `$commit-message Draft a message for my staged changes.` |
| Claude Code | `/commit-message Draft a message for my staged changes.` |

Expect one copyable message describing only the staged changes. The skill does
not stage files or create a commit. Compare the message with `git diff --cached`
before using it.

Nothing staged? Try the [writing example](docs/examples/concise-output.md).
See the [commit-message example](docs/examples/commit-message.md) for a sample
result, or [browse examples by task](EXAMPLES.md).

## Highlights

- **Clear writing:** plain English answers and documentation, with details
  introduced gradually.
- **Coding standards:** guidance for Python, JavaScript, TypeScript, Go,
  Ruby/Rails, and Bash.
- **Planning:** feasibility checks, tickets, and implementation plans.
- **Implementation:** one planned unit or a full phase at a time.
- **Review:** code review, security assessment, and manual test cases.

## Skill catalog

| Category | Pack | Purpose |
| --- | --- | --- |
| coding | [bash-standards](.agents/skills/coding/bash-standards/SKILL.md) | Routed Bash engineering guidance. |
| coding | [define-coding-standards](.agents/skills/coding/define-coding-standards/SKILL.md) | Build a language standards pack. |
| coding | [golang-standards](.agents/skills/coding/golang-standards/SKILL.md) | Routed Go engineering guidance. |
| coding | [javascript-standards](.agents/skills/coding/javascript-standards/SKILL.md) | Routed JavaScript engineering guidance. |
| coding | [python-standards](.agents/skills/coding/python-standards/SKILL.md) | Routed Python engineering guidance. |
| coding | [ruby-standards](.agents/skills/coding/ruby-standards/SKILL.md) | Routed Ruby engineering guidance. |
| coding | [ruby-rails-standards](.agents/skills/coding/ruby-rails-standards/SKILL.md) | Routed Ruby on Rails engineering guidance. |
| coding | [typescript-standards](.agents/skills/coding/typescript-standards/SKILL.md) | Routed TypeScript engineering guidance. |
| implementation | [plan-execution-conventions](.agents/skills/implementation/plan-execution-conventions/SKILL.md) | Interpret existing plans, validate completion, and track execution progress. |
| implementation | [next-commit](.agents/skills/implementation/next-commit/SKILL.md) | Implement one planned unit without committing. |
| implementation | [next-phase](.agents/skills/implementation/next-phase/SKILL.md) | Validate and commit each implementation unit locally; keep planning progress uncommitted unless requested. |
| other | [developer-highlights](.agents/skills/other/developer-highlights/SKILL.md) | Evidence-based career narrative from git history. |
| productivity | [commit-message](.agents/skills/productivity/commit-message/SKILL.md) | Short commit message from staged changes and available conversation context. |
| productivity | [concise-output](.agents/skills/productivity/concise-output/SKILL.md) | Readable, concise prose that explains the outcome before the details. |
| productivity | [pr-description](.agents/skills/productivity/pr-description/SKILL.md) | Copy-ready pull-request description. |
| productivity | [pr-comment](.agents/skills/productivity/pr-comment/SKILL.md) | Concise, human response to pull-request feedback. |
| productivity | [to-clipboard](.agents/skills/productivity/to-clipboard/SKILL.md) | Copy the previous response to the clipboard as Markdown. |
| productivity | [to-markdown](.agents/skills/productivity/to-markdown/SKILL.md) | Save the previous response as a formatted Markdown file. |
| review | [plan-review](.agents/skills/review/plan-review/SKILL.md) | Review plan readiness . |
| review | [annotate](.agents/skills/review/annotate/SKILL.md) | Explain branch changes in terminal output without modifying files. |
| review | [review-branch](.agents/skills/review/review-branch/SKILL.md) | Evidence-based review of committed branch changes. |
| scoping | [plan-writing-conventions](.agents/skills/scoping/plan-writing-conventions/SKILL.md) | Define plan structure, scope, validation, and risks. |
| scoping | [quick-plan](.agents/skills/scoping/quick-plan/SKILL.md) | Investigate feasibility and give one recommendation without implementation or a commit plan. |
| scoping | [requirements-to-tickets](.agents/skills/scoping/requirements-to-tickets/SKILL.md) | Turn feature briefs into small, self-contained tickets. |
| scoping | [ticket-context](.agents/skills/scoping/ticket-context/SKILL.md) | Build a repository-backed context brief for a ticket. |
| scoping | [ticket-to-plan](.agents/skills/scoping/ticket-to-plan/SKILL.md) | Turn a file, chat, or Jira ticket into an implementation-ready planning folder. |
| scoping | [test-cases](.agents/skills/scoping/test-cases/SKILL.md) | Create prioritized Markdown manual test cases for branch changes. |
| security | [security-check](.agents/skills/security/security-check/SKILL.md) | Short fix, accept, or monitor decision. |
| security | [security-review](.agents/skills/security/security-review/SKILL.md) | Detailed security assessment. |

See [CONTRIBUTING.md](CONTRIBUTING.md) for layout and validation rules.

## Development workflows

Skills work independently or pass results to one another. Use a short
combination for a focused task, or take a ticket through planning,
implementation, validation, and review.

See [skill combinations and development workflows](docs/usage/workflows.md)
for handoffs, optional steps, and the choice between implementing one unit
with `next-commit` or a phase with `next-phase`.

## Guides

See [INSTALL.md](INSTALL.md) for local setup and maintenance, or
[native packages](docs/usage/native-packages.md) for provider plugins. The
[documentation index](docs/README.md) covers setup, plugin use, and publishing.
See the [repository overview](docs/overview.md) for how the shared skill library,
installation routes, and development workflow fit together.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for repository rules. Run `make verify`
to check the catalog, packages, and isolated installations for both providers.
