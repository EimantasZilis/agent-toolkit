# agent-toolkit guidance

For code edits or reviews, load the appropriate installed coding standards pack
such as `python-standards` and route to only the references that apply.

Before taking action, read the pack's complete `SKILL.md`. Then read every
reference it marks as always required and every specialist reference selected
by the changed context. State the exact references loaded before editing or
reviewing code; do not proceed silently when a required reference is missing
or unreadable.

Repository-specific guidance wins.

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
