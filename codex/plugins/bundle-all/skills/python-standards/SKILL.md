---
name: python-standards
description: Apply the repository's Python engineering guidance to Python edits and reviews, routing only the relevant references for core code, tests, frameworks, errors, docs, and logging.
---

# Python standards

Before editing or reviewing, read this complete `SKILL.md`, then load every
reference selected below. Always load [core](references/core.md) and
[linting](references/linting.md); add only the context-specific guidance
below. State the exact references loaded before taking action.

| Change context | Load |
| --- | --- |
| Tests | [tests](references/tests.md) |
| Django or DRF | [web framework](references/web-framework.md), [imports](references/imports.md) |
| Imports or package boundaries | [imports](references/imports.md) |
| Exceptions, retries, or fallbacks | [errors](references/errors.md) |
| Changed behavior, docstrings, or explanatory comments | [docstrings](references/docstrings.md) |
| Logging | [logging](references/logging.md) |

Apply repository-specific rules when they are stricter or more relevant. Keep
this pack additive, report only violations visible in changed code, and do not
rewrite unrelated lines. Keep functions focused, respect the repository's
formatter and linter, avoid unnecessary dependencies, and verify changed paths
plus relevant tests. Never log secrets or automatically commit changes.

Apply the pack in two passes. First run the repository's mechanical formatter
and linter. Then perform an LLM review-and-edit pass over the changed code for
guidance that tools cannot infer, including function responsibility, semantic
variable grouping, logical phase boundaries, vertical spacing, docstrings,
comments, and test structure. Do not treat formatter or linter success as
completion of this pass.

Treat the applicable checklists as completion checks, not background reading.
Use the `DO` examples as the expected shape and an allowed exception only when
the reference names a concrete reason. If a preferred pattern is not suitable,
state the local constraint or design reason in the review rather than silently
dropping the pattern.

Before finalizing, inspect the complete changed diff after formatting. Add
blank lines between related variable groups and distinct phases inside
functions when they improve scanability; do not add a blank line after every
statement or infer arbitrary phases in a short linear function. Preserve
unrelated existing code unless the user requests a broader standards cleanup.

For test changes, do not omit the test structure: use blank lines to separate
setup, act, and assert, and add a concise GIVEN/WHEN/THEN docstring whenever
the test covers branching, persistence, I/O, an external boundary, multiple
meaningful setup steps, or an integration-like flow. Before finalizing, inspect
the changed diff for these requirements because formatters do not enforce them.
