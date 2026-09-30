# Contributing to agent-toolkit

Skills live at `.agents/skills/<category>/<skill-name>/SKILL.md`. The front
matter `name` must match the skill directory, and `description` must state the
trigger. Keep this source tree provider-neutral. Codex metadata belongs under
`.agents/providers/codex/`; Claude invocation metadata belongs under
`.agents/providers/claude/`.

Global guidance lives at `.agents/global/instructions.md`. Select the target
provider with `LLM_PROVIDER=codex` or `LLM_PROVIDER=claude` in `.env` (or an
exported environment variable), then run the installer. Local Markdown links
are relative to their source file.

Before changing a skill, preserve its public behavior, update the catalog, and
run `make verify`. Keep references in Markdown rather than adding editor
rules, routers, or unsupported configuration.
