# Provider-neutral coding-standards pack layout

```text
.agents/skills/coding/<lang>-standards/
  SKILL.md
  references/
    core.md
    linting.md
    <lang>-tests.md
    <lang>-<topic>.md
```

## Canonical contract

The `SKILL.md` file is a concise index for both coding edits and code reviews.
It must:

- keep the pack name lowercase and use matching `name` metadata;
- state source precedence, with repository-local rules winning when they are
  stricter or more relevant;
- always load `core.md` and `linting.md`;
- route specialist references only for the changed context through a compact
  decision table;
- link every reference it names with a relative Markdown link;
- define validation of changed paths, relevant tests, and the repository's
  formatter/linter/verification commands; and
- retain explicit invocation metadata when the pack is intended to be
  explicitly invoked.

Core covers the language's baseline implementation and review rules. Linting
covers formatting, linting, dependency, documentation, secret-handling, and
verification expectations. Specialist references cover only distinct contexts
such as tests, frameworks, errors, logging, migrations, or documentation.

Every reference must contain actionable rules, an `Apply this first` or
equivalent entry section, a `Quick checklist`, `DO` and `DO NOT` examples, and
documented concrete exceptions. Examples must show the preferred shape rather
than merely name a rule. Checklists are completion checks for edits and
reviews, not background reading.

The index routes the agent to the smallest relevant references: baseline files
are mandatory and specialist files are conditional. Do not make Python-only
assumptions in this contract; the Python pack is the structural reference.

Keep provider invocation policy in the relevant adapter under
`.agents/providers/`. Do not add provider-specific files to the source pack,
or add Cursor-specific files, globs, routers, or unsupported configuration.
