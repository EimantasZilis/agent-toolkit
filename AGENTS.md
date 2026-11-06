# Agent guide — agent-toolkit

This repository ships reusable skills for Codex and Claude. Provider-neutral
skills live in `.agents/skills/`, provider adapters in `.agents/providers/`, and
the optional global guidance template in `.agents/global/instructions.md`.

Local setup uses the Bash installation backend in `scripts/`. The Go TUI in
`cmd/agent-toolkit/` and `internal/` is another client of that backend. Native
package tooling builds self-contained provider plugins from the same skills.
See the [documentation index](docs/README.md) for usage and maintenance
guides.

Before changing a skill, read [CONTRIBUTING.md](CONTRIBUTING.md). Keep skill
names stable, use Markdown references rather than editor-specific rules, and
preserve relative links. Skills must remain compatible with both Codex and
Claude; follow the provider-neutral source format and provider-specific adapter
rules in [CONTRIBUTING.md](CONTRIBUTING.md). When adding or editing skills,
keep [README.md](README.md) up to date with the skills and their summaries.

Run `make verify` before reporting a structural change complete.
This covers skills, tooling, provider adapters, and installation contracts.
Generating plans or editing ordinary documentation does not trigger it; use
artifact checks unless a specific claim needs an executable check.
