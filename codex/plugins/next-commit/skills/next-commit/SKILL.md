---
name: next-commit
description: Implement exactly the next unchecked unit in a phase-indexed plan, validate it, and update planning progress without creating a git commit.
---

# Next commit

For optional integrations, use installed skills and their required references;
otherwise silently follow local defaults, repository conventions, and configured
checks. Do not warn about unavailable optional skills or request their
installation. Explicit repository requirements remain binding.

Use the planning README as the sole entrypoint. Resolve an explicit plan path
first, otherwise use the current branch's managed `ticket-to-plan` output
directory and then legacy `docs/planning/`. Use `plan-execution-conventions`
when available. Otherwise the README indexes phase files, [ ] means pending,
[-] active, and [x] complete; units are ordered, scoped changes with validation.

Resolve missing phase files, ambiguous status, scope, or validation before
implementation. If no unchecked work remains, report that without creating
another unit. The plan's producer skill does not need to be installed.

1. Find the active phase ([-], otherwise first [ ]) and open its phase file.
2. Check the current branch without switching; pause on main or master.
3. Before editing code, inspect the planned implementation paths and apply
   relevant language standards.
4. Implement only the first unchecked commit, with no scope expansion. Include
   formatting changes in the unit and rerun validation after any changes.
5. Run that commit's validation, or the phase exit checks if none is defined.
   When finishing a phase, also run its exit checks. Stop on validation failure
   or cancellation, preserving completed work and accurate progress.
6. Mark the commit complete only after validation passes. Update the phase
   status in the README; mark it complete only when every unit and exit check
   has passed.
7. Keep ticket and planning Markdown changes as local progress by default; do
   not include managed output artifacts, `docs/planning/**`, `docs/jira/**`, or
   other ticket/plan files in the suggested commit unless the user explicitly
   asks. Report changes, a suggested message, implementation files, and the
   next action. Do not run git commit, push, or continue to another unit.

If the plan is materially wrong, stop and propose a docs-first amendment. When a
phase completes, point to `review-branch` (optional integration)
and `pr-description` (optional integration); otherwise
point to `next-phase` (optional integration).

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

### Native output policy

Honor explicit user guidance configuring an output location, including a centralized output root. Otherwise use `<repo-root>/<branch-slug>/<skill-name>/...`. These task-local defaults do not modify global settings.
