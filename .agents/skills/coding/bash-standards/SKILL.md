---
name: bash-standards
description: Apply the repository's Bash engineering guidance to Bash edits and reviews, routing only the relevant references for core code, linting, errors, tests, and security-sensitive scripts.
---

# Bash standards

Load the references needed for the change before editing. Always load
[core](references/core.md) and [linting](references/linting.md); add only the
context-specific guidance below.

| Change context | Load |
| --- | --- |
| Exit status, traps, cleanup, retries, or fallback behavior | [errors](references/errors.md) |
| Bats, shell integration, fixtures, or command-output assertions | [tests](references/tests.md) |
| Untrusted input, temporary files, privileges, secrets, or destructive commands | [security](references/security.md) |

Apply repository-specific rules when they are stricter or more relevant. The
precedence is: repository-local rules, official Bash documentation, then
established guidance from Google Shell Style, ShellCheck, shfmt, and Bats.
This pack applies to both implementation and review. Keep scripts small and
focused, quote expansions by default, make exit status intentional, avoid
unnecessary dependencies, and do not rewrite unrelated lines.

Validate changed paths, run the repository's formatter, linter, and relevant
tests when configured, and run the repository verification command. In this
repository, run `make verify`; for Bash files also run `bash -n`, ShellCheck,
and shfmt when those tools are available or prescribed locally. Never log
secrets or automatically commit changes.

Treat each applicable checklist as a completion check. Use the `DO` examples
as the expected shape and an allowed exception only when the reference names a
concrete reason. If a preferred pattern is unsuitable, record the local
constraint or design reason in the review.
