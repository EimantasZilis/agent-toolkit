# Agent guide — agent-toolkit

This repository ships agent skills from `.agents/skills/` and a managed global
guidance template from `.agents/global/AGENTS.md`. There is no application
runtime.

Before changing a skill, read [CONTRIBUTING.md](CONTRIBUTING.md). Keep skill
names stable, use Markdown references rather than editor-specific rules, and
preserve relative links. Skills must remain compatible with both Codex and
Claude; follow the provider-neutral source format and provider-specific adapter
rules in [CONTRIBUTING.md](CONTRIBUTING.md).

Run `make verify` before reporting a structural change complete.
