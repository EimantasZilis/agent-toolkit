---
name: javascript-standards
description: Apply provider-neutral JavaScript engineering guidance to JavaScript edits and reviews, routing only the relevant references for core code, modules, tests, errors, and documentation.
---

# JavaScript standards

Apply repository-local rules first when they are stricter or more relevant;
then apply official ECMAScript/runtime documentation, followed by the
referenced community tooling guidance. This pack applies to both edits and
reviews. Always load [core](references/core.md) and
[linting](references/linting.md), then route only the changed context below.

| Change context | Load |
| --- | --- |
| ESM, CommonJS, package boundaries, or imports | [modules](references/modules.md) |
| Promises, async code, exceptions, retries, or cleanup | [errors](references/errors.md) |
| Unit, integration, or browser tests | [tests](references/tests.md) |
| Public APIs, comments, or JSDoc | [documentation](references/documentation.md) |

Validate changed paths with the repository's formatter, linter, type checker
if present, and relevant tests. Do not introduce a formatter or framework
without repository justification, and do not log secrets. Treat each linked
checklist as a completion check; use an exception only when its concrete
reason is documented in the change or local configuration.
