# Agent guide — agent-toolkit

This repository ships reusable skills for Codex and Claude. Provider-neutral
skills live in `.agents/skills/`, provider adapters in `.agents/providers/`, and
the optional global guidance template in `.agents/global/instructions.md`.

Local management runs through the Go TUI in `cmd/agent-toolkit/` and the typed
installation service under `internal/install/`. Shared catalog and rendering
packages also support self-contained native plugins. Start local management
with `make run` or `go run ./cmd/agent-toolkit`; scripted local management has
been removed. Native build and validation commands are maintainer tools.
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
Run `make smoke-source` when changing startup or the TUI. It exercises actual
terminal startup from current source in isolated homes. CI runs both checks
on macOS and Linux. Go and Git are tooling prerequisites; Make is optional
for local management but required by the source smoke check.

For plans or ordinary documentation, use artifact checks unless a specific
claim needs an executable check.
