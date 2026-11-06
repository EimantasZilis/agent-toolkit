---
name: ruby-rails-standards
description: Extend the provider-neutral Ruby baseline with Rails engineering guidance for Rails edits and reviews, routing only the relevant framework references.
---

# Ruby on Rails standards

This is a Rails extension pack. Load [ruby-standards](../ruby-standards/SKILL.md)
first for the language baseline. Always load this pack's [core](references/core.md)
and [linting](references/linting.md) overlays; add only the context-specific
Rails references below.

| Change context | Load |
| --- | --- |
| Controllers, models, views, routes, autoloading, service objects, or Rails conventions | [Rails application](references/rails-application.md) |
| Migrations, schemas, queries, validations, transactions, or callbacks | [data](references/data.md) |
| Rails Minitest, RSpec, request, integration, job, or system tests | [tests](references/tests.md) |
| Exceptions, retries, jobs, transactions, or fallback behavior | [errors](references/errors.md) |
| Parameters, authorization, secrets, HTML, SQL, or user-controlled input | [security](references/security.md) |
| Logs, notifications, metrics, tracing, or audit events | [logging](references/logging.md) |
| Public APIs, user-facing behavior, comments, or API documentation | [documentation](references/documentation.md) |

Apply repository-local rules when they are stricter or more relevant. The
precedence is: repository-local rules and configuration; official Ruby and
Rails documentation; then established Ruby/Rails community guidance. This
pack applies to both implementation and review. Keep changes focused, preserve
existing public behavior unless the task changes it, never log secrets, and do
not automatically commit changes. Do not load Rails references for standalone
Ruby gems, scripts, or libraries; use `ruby-standards` alone there.

Validate changed paths, run the repository's configured formatter and linter,
run relevant tests, and run the repository verification command. For Rails
applications, prefer `bin/rails` and the checked-in Bundler lockfile; use the
project's configured RuboCop/RuboCop Rails and test commands rather than
inventing versions or flags.

Treat each applicable checklist as a completion check. Use the `DO` examples as
the expected shape and use an allowed exception only when its reference names
a concrete reason. Report only violations visible in changed code during
reviews.
