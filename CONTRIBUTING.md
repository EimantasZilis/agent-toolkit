# Contributing to agent-toolkit

See the [repository overview](docs/overview.md) for how the shared skill library,
local installation, and plugin publication fit together.

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
installation. Choose providers and guidance in the TUI started by `make run`
or `go run ./cmd/agent-toolkit`. Local Markdown links are relative to their
source file; rendering resolves cross-skill links for the installed layout.

Declare only required in-session skill loads and invocations in
`.agents/dependencies.txt`: ticket planning requires plan review, and Rails
standards require the Ruby baseline. Dependencies are installed transitively
and deduplicated. Keep this registry synchronized when changing a workflow.
Do not include optional integrations, entry-point routing, or recommendations
for later workflows. The detailed optional-integration policy lives in
`.agents/global/instructions.md`. Each skill with optional integrations must
include one compact, self-contained rule covering all of them, so standalone
installation does not require global guidance. Keep only skill names and
task-specific fallback behaviour at individual use sites; do not repeat the
general policy for every integration. Explicit repository requirements and
mandatory helpers remain binding.

`plan-writing-conventions` owns plan authoring, while
`plan-execution-conventions` owns interpreting plans and tracking execution.
Keep creation and execution skills usable without those optional conventions.
Maintain compatibility through the plan format and artifact discovery rules,
not a required installation of the plan's producer skill.

Markdown-producing or discovery workflows also need task-local output rules;
maintain the shared consumer registry in
`.agents/markdown-output-skills.txt` when adding one. Both local installation
and native packaging read this registry; each entry must name an existing skill.
Validate selective and full-catalog installation for both providers with `make verify`.

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

## Skill examples

Keep examples in `docs/examples/` focused on the skill's value, an example
prompt, and the result it produces. Include a `Why use it` section in every
example. Use one short paragraph to explain the practical benefit beyond the
sample scenario. Aim for two or three sentences without repeating the
introduction or result. Ground benefits in the skill's actual behavior and avoid
unsupported promises. Apply this structure to new and existing examples.

Do not add disclaimers about examples being illustrative or commands and
application checks not being executed. This applies to both new and existing
examples.

## Source application checks

Repository tooling uses Go and Git. Go uses the version declared in `go.mod`.
The TUI calls the typed installation service directly. Catalog and rendering
packages are shared by local installation and native packaging.

| Check | Make command | Go command |
| --- | --- | --- |
| Source metadata, links, layout and registries | `make validate` | `go run ./cmd/agent-toolkit validate` |
| Package tests | `make test` | `go test ./...` |
| Tests, vet and isolated provider installations | `make verify` | Run tests, `go vet ./...`, then `go run ./cmd/agent-toolkit verify` |
| Actual TUI startup from current source | `make smoke-source` | `go run ./cmd/agent-toolkit smoke-source` |

Run `make verify` and `make smoke-source` when changing setup or the TUI.
The smoke check exercises `make run`, so that check requires Make even when
invoked through Go. Local management and native commands can run without Make.
Smoke runs in a spaced checkout, isolates provider homes, and verifies read-only
quit after a source change. CI runs both commands on macOS and Linux with
read-only repository permissions.

Tests live beside their Go packages, with fixtures under `testdata/`.
The bundled Python checker remains a skill resource. Repository checks inspect
its contents, modes, and references without executing it. Using that skill's
helper can still require Python.

See the [verification coverage map](docs/maintenance/verification.md) for behavior
preserved from earlier tooling. Native package builds are independent of local
installation; follow the [build](docs/maintenance/build-native.md) and
[manual publication](docs/maintenance/publish-native.md) guides.
