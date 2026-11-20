# Markdown output settings

Apply these instructions when saving or finding generated Markdown files. Use
the Markdown artifact location in the global settings block when the user does
not provide an explicit path. An explicit user path always wins. Standalone
native packages supply a fallback when no global location is configured.
Changing storage leaves existing artifacts in place.

## Markdown artifact outputs

When a skill writes a Markdown artifact and the user has not supplied an
explicit path, use the installed output-location policy. In repo-local mode,
write beneath `<repo-root>/<skill-name>/<branch-slug>/`; in external mode,
write beneath `<configured-output-root>/<repo-slug>/<skill-name>/<branch-slug>/`.
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
   branch: `<repo-root>/<producer-skill>/<branch-slug>/`.
3. The configured centralized output directory for the current repository and
   branch: `<configured-output-root>/<repo-slug>/<producer-skill>/<branch-slug>/`.
4. Previous branch/skill layouts beneath the same repository or configured
   output root.
5. A documented legacy location such as `docs/planning/`, when supported.

At each producer-skill/branch location, consider artifacts directly in the
shared directory and in its `codex/` and `claude/` subdirectories together,
regardless of the current separation setting. Do not prefer the current
provider. Ask the user to choose when multiple plausible artifacts match.
Explicit output guidance may enable a provider suffix for new artifacts;
otherwise keep the output directory shared.

Determine `<repo-root>` with the target repository's Git root, not the current
working directory. Determine the branch from that repository; detached HEADs
use `detached-<short-sha>`. Search only the producer skill's directory, keep
the current repository and branch in scope, and ask the user to choose when
multiple plausible artifacts remain. Do not silently search another repository,
branch, or output root.

Skills that form a workflow must name their expected producer skill and use
this discovery order before falling back to repository-wide searches.
