# Coding-standards critique checklist

- [ ] `SKILL.md` has matching `name` metadata and a trigger-focused description.
- [ ] Pack and reference names are lowercase, and explicit invocation metadata is preserved when required.
- [ ] Index routing links every reference that it names with a valid relative Markdown link.
- [ ] The index always routes the core and linting baseline references.
- [ ] A compact decision table routes each specialist reference only when its context is relevant.
- [ ] The index explicitly covers both edits and reviews, changed-path validation, and relevant tests or verification.
- [ ] Source precedence is stated, with repository-local rules winning when stricter or more relevant.
- [ ] Core and linting references cover baseline implementation and review checks.
- [ ] Every specialist reference is actionable and includes an apply-first section, quick checklist, `DO` and `DO NOT` examples, and concrete documented exceptions.
- [ ] Guidance is backed by authoritative sources, concise, and compatible with existing local rules.
- [ ] No Cursor-specific files, metadata, paths, or configuration are introduced.
- [ ] README, contributor guidance, and changelog are updated only when the public inventory or repository contract changes.
- [ ] `make verify` passes.
