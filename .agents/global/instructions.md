# agent-toolkit guidance

For code edits or reviews, use relevant installed coding standards packs such
as `python-standards` when available. Otherwise follow repository conventions
and configured checks.

Repository-specific guidance wins.

## Optional skill integrations

For optional integrations, use only skills available in the active environment's
installed skill catalog. Read each skill's complete instructions and its required
references before applying it; load specialist references only for the relevant
context. State the exact references loaded before editing or reviewing code.
Do not load uninstalled skills from the toolkit source tree.

When an optional skill is absent, silently use the workflow's local defaults,
repository guidance, existing code conventions, configured formatters and
linters, and relevant validation. Do not warn about unavailable optional skills
or ask users to install them. A skill's availability does not authorize extra
actions, such as running checks during a static review.

Explicit repository requirements and mandatory skill dependencies remain
binding. An installed skill's required references are not optional: report
missing or unreadable required references rather than silently falling back.

## Code readability

For every language and coding standards pack, write human-readable code. Use
blank lines to separate logical groups of variable declarations, expressions,
and statements, and to distinguish blocks or stages of work. Keep closely
related lines together; avoid both dense code and unnecessary blank lines.
Follow the language's conventions and the repository's formatter, and review
the final code for readable spacing before completing implementation.
