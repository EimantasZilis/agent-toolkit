---
name: define-coding-standards
description: Create or extend a provider-neutral coding-standards pack for one language from authoritative sources; explicit invocation only.
---

# Define coding standards

Read CONTRIBUTING.md, pack-structure.md, and review-lens.md before editing.

Use this workflow when creating or extending a pack:

1. Inspect the target repository's AGENTS.md files, contributor guidance,
   formatter/linter/test configuration, and existing standards packs. Treat an
   existing pack as a structural example, not as an authority for the new
   language.
2. Research the language and the contexts in scope before writing rules. Start
   with official language, framework, and standards documentation, then search
   for established style guides and engineering-practice guidance from
   reputable sources. Use the full source rather than search-result snippets,
   and gather multiple independent sources for each substantial topic.
3. Compare the sources before adopting advice. Give extra confidence to rules
   repeated across independent sources, but do not treat repetition as proof:
   check that the sources are relevant, current, and technically independent.
   Prefer a directly applicable official rule over community consensus. When
   sources conflict, apply this precedence: repository-local rules and
   conventions; official language, framework, or standards documentation; then
   well-established community guidance. Record the decision and a concrete
   exception when a local rule intentionally differs from an upstream source.
   Do not invent a consensus from a single source or from copied
   recommendations.
4. Record traceability in the generated pack. Link the authoritative sources
   that support each substantial rule, identify where multiple sources agree,
   and briefly document material disagreements or judgment calls. Prefer
   stable canonical URLs and note access dates when guidance is likely to
   change.
5. Create a lowercase `<lang>-standards` pack with a concise index, mandatory
   `core.md` and `linting.md` references, and specialist references for
   tests, frameworks, errors, documentation, logging, data, or other
   contexts the language needs.
6. Make the index route only relevant specialist references while always
   loading core and linting. State the source precedence, local-rule override,
   and that the pack applies to both edits and reviews. Keep the routing table
   small enough to choose references from the changed context.
7. Make every reference actionable: begin with application rules, include a
   quick completion checklist, show `DO` and `DO NOT` examples, and document
   concrete allowed exceptions. Cover both implementation and review decisions.
8. Validate the generated pack against review-lens.md. Check every relative
   link and required file, preserve lowercase naming and explicit invocation
   metadata, run the repository verification command, and inspect the final
   diff for unrelated changes.

Do not create Cursor rules, globs, routers, or a configuration file. Keep the
contract language-agnostic: Python is the reference implementation, not a
special case.
