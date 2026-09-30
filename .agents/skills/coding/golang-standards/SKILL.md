---
name: golang-standards
description: Apply provider-neutral Go engineering guidance to Go edits and reviews, routing only the relevant references for core code, tooling, tests, errors, concurrency, documentation, and logging.
---

# Go standards

Apply repository-local rules and tool configuration first when they are
stricter or more relevant; then apply official Go documentation and established
community guidance recorded in the references. This pack applies to both edits
and reviews. Always load [core](references/core.md) and
[linting](references/linting.md), then route only the changed context below.

The official Go documentation, Effective Go, and Go Code Review Comments
converge on formatting, simple package design, explicit context propagation,
documented exported APIs, and deliberate error handling. Where local tooling
or an older supported Go version differs from newer library guidance (for
example, `log/slog`), follow the repository's supported version and record
the compatibility reason.

| Change context | Load |
| --- | --- |
| Tests, benchmarks, examples, or fuzzing | [tests](references/tests.md) |
| Errors, retries, cleanup, or process boundaries | [errors](references/errors.md) |
| Goroutines, channels, shared state, or cancellation | [concurrency](references/concurrency.md) |
| Exported APIs, package comments, or examples | [documentation](references/documentation.md) |
| Logs, metrics, tracing, or audit events | [logging](references/logging.md) |

Validate changed paths with the repository's `gofmt`, configured linter, and
`go vet` or equivalent; run relevant `go test` packages (with `-race` when
concurrency is affected) and the repository verification command. Run module
and vulnerability checks when dependencies change. Treat each applicable
checklist as a completion check and document concrete exceptions in the change
or repository configuration. Do not commit automatically.
