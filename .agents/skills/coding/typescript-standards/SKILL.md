---
name: typescript-standards
description: Apply provider-neutral TypeScript engineering guidance to TypeScript edits and reviews, routing only the relevant references for core types, compiler and linting, modules, tests, errors, and documentation.
---

# TypeScript standards

Apply repository-local rules first when they are stricter or more relevant;
then apply official TypeScript, ECMAScript, runtime, and referenced tooling
documentation. This pack applies to both edits and reviews. Always load
[core](references/core.md) and [linting](references/linting.md), then route only
the changed context below.

| Change context | Load |
| --- | --- |
| Module boundaries, tsconfig, packages, or declaration output | [modules](references/modules.md) |
| Exceptions, async code, retries, or resource cleanup | [errors](references/errors.md) |
| Unit, integration, or browser tests | [tests](references/tests.md) |
| Public APIs, JSDoc, or explanatory comments | [documentation](references/documentation.md) |

Validate changed paths with the repository's formatter, linter, tsc or
equivalent type checker, and relevant tests. Treat each linked checklist as a
completion check; use an exception only when its concrete reason is documented
in the change or local configuration. Do not use types as a substitute for
runtime validation of untrusted data.
