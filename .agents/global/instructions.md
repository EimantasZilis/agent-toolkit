# agent-toolkit guidance

For code edits or reviews, load the appropriate installed coding standards pack
such as `python-standards` and route to only the references that apply.

Before taking action, read the pack's complete `SKILL.md`. Then read every
reference it marks as always required and every specialist reference selected
by the changed context. State the exact references loaded before editing or
reviewing code; do not proceed silently when a required reference is missing
or unreadable.

Repository-specific guidance wins.

## Code readability

For every language and coding standards pack, write human-readable code. Use
blank lines to separate logical groups of variable declarations, expressions,
and statements, and to distinguish blocks or stages of work. Keep closely
related lines together; avoid both dense code and unnecessary blank lines.
Follow the language's conventions and the repository's formatter, and review
the final code for readable spacing before completing implementation.

## Markdown artifact outputs

When a skill writes a Markdown artifact and the user has not supplied an
explicit path, use the installed output-location policy. In repo-local mode,
write beneath `<repo-root>/<branch-slug>/<skill-name>/`; in external mode,
write beneath `<configured-output-root>/<repo-slug>/<branch-slug>/<skill-name>/`.
Create slugs by lowercasing the repository directory name or branch name,
replacing each run of characters outside `a-z`, `0-9`, `.`, `_`, and `-` with
`-`, trimming leading and trailing `-` characters, and using `repo` or
`branch` when the result is empty. For detached HEADs, use
`detached-<short-sha>`. Keep all files from one invocation under its skill
directory, avoid overwriting existing artifacts, and report the exact path.
An explicit user-provided path always wins. This policy applies to existing
skills and every future skill that writes Markdown.

## Markdown artifact discovery

When a skill consumes Markdown produced by another skill, resolve artifacts in
this order:

1. An explicit path supplied by the user or referenced by the current task.
2. The current Git repository's managed output directory for the current
   branch: `<repo-root>/<branch-slug>/<producer-skill>/`.
3. The configured centralized output directory for the current repository and
   branch: `<configured-output-root>/<repo-slug>/<branch-slug>/<producer-skill>/`.
4. A documented legacy location such as `docs/planning/`, when supported.

Determine `<repo-root>` with the target repository's Git root, not the current
working directory. Determine the branch from that repository; detached HEADs
use `detached-<short-sha>`. Search only the producer skill's directory, keep
the current repository and branch in scope, and ask the user to choose when
multiple plausible artifacts remain. Do not silently search another repository,
branch, or output root.

Skills that form a workflow must name their expected producer skill and use
this discovery order before falling back to repository-wide searches.
