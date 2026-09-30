---
name: ruby-standards
description: Apply provider-neutral Ruby engineering guidance to Ruby edits and reviews, routing only the relevant references for core code, linting, tests, errors, security, logging, and documentation.
---

# Ruby standards

Load the references needed for the change before editing. Always load
[core](references/core.md) and [linting](references/linting.md); add only the
context-specific guidance below.

| Change context | Load |
| --- | --- |
| Minitest, RSpec, unit tests, fixtures, or test doubles | [tests](references/tests.md) |
| Exceptions, cleanup, retries, or fallback behavior | [errors](references/errors.md) |
| User-controlled input, serialization, credentials, or untrusted data | [security](references/security.md) |
| Logs, metrics, tracing, or audit events | [logging](references/logging.md) |
| Public APIs, executable behavior, comments, or library documentation | [documentation](references/documentation.md) |

For Rails applications, load [ruby-rails-standards](../ruby-rails-standards/SKILL.md)
in addition to this baseline. Apply repository-local rules when they are
stricter or more relevant. The precedence is: repository-local rules and
configuration; official Ruby documentation; then established Ruby community
guidance. This pack applies to both implementation and review.

Keep changes focused, preserve public behavior unless the task changes it,
avoid unnecessary dependencies, never log secrets, and do not automatically
commit changes. Validate changed paths, run the repository's configured
formatter, linter, relevant tests, and verification command. Prefer
Bundler-scoped tools and the checked-in lockfile.

Treat each applicable checklist as a completion check. Use the `DO` examples as
the expected shape and use an allowed exception only when its reference names a
concrete reason. Report only violations visible in changed code during reviews.

