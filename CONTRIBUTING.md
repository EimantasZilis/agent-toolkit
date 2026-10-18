# Contributing to agent-toolkit

Skills live at `.agents/skills/<category>/<skill-name>/SKILL.md`. The front
matter `name` must match the skill directory, and `description` must state the
trigger. Keep this source tree provider-neutral. Codex metadata belongs under
`.agents/providers/codex/`; Claude invocation metadata belongs under
`.agents/providers/claude/`.

## Cross-provider skill compatibility

Every source skill under `.agents/skills/` must be usable by both Codex and
Claude. Keep `SKILL.md` provider-neutral: use standard Markdown and YAML front
matter, keep `name` and the trigger-focused `description` valid for both
providers, use relative Markdown links to files within the skill, and do not
include provider-specific metadata, paths, tools, or syntax.

Put provider-specific behavior in `.agents/providers/<provider>/` and have the
installer render or attach it for that provider. When adding or changing a
skill, run `make verify`; verification must cover installation for both Codex
and Claude.

Codex adapters belong at
`.agents/providers/codex/skills/<category>/<skill-name>/agents/openai.yaml`;
Claude explicit-only entries use the same `<category>/<skill-name>` key.
Adapters are optional, but every adapter and explicit-only entry must resolve
to a source skill. Move or remove adapters when moving or removing a skill;
`make validate` checks these references before `make verify` checks both
providers' installed metadata.

Global guidance lives at `.agents/global/instructions.md` and is optional at
installation. Select the provider in `.env` or pass `--provider codex` or
`--provider claude` to `scripts/install.sh`. Local Markdown links are relative
to their source file; rendering resolves cross-skill links for the installed
layout.

Declare required in-session skill loads and invocations in
`.agents/dependencies.txt`, including conditional standards and security gates.
Do not include entry-point routing or recommendations for later workflows.
Dependencies are installed transitively; conditional cycles such as Ruby/Rails
are deduplicated. Keep this registry synchronized when changing a workflow.
Markdown-producing or discovery workflows also need task-local output rules;
maintain the consumer list in `scripts/skills.sh` when adding one. Validate
selective and full-catalog installation for both providers with `make verify`.

## Markdown artifact outputs

Skills that write Markdown must follow the installed output-location policy.
When no path is supplied, write beneath the current repository's branch and
skill directory in repo-local mode, or beneath the configured
`<output-root>/<repo>/<branch>/<skill>/` hierarchy in external mode. Explicit
user-provided paths always win. Keep all files from one invocation beneath the
skill directory, avoid overwriting existing artifacts, and report the exact
written path. Do not hardcode `docs/` as the default destination for new
skills.

Before changing a skill, preserve its public behavior, update the catalog, and
run `make verify`. Keep references in Markdown rather than adding editor
rules, routers, or unsupported configuration.

When adding or amending skills, keep the skill summaries in `README.md`
grouped by category, with categories ordered alphabetically. Within each
category, follow the workflow order where one exists; otherwise, order skills
alphabetically by name. Put shared guidance before the workflow steps it
supports, and order independent skills alphabetically.
