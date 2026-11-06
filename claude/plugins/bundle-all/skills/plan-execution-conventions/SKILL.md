---
name: plan-execution-conventions
description: Interpret phase-indexed plans, select implementation units, validate completion, and update progress when executing an existing plan.
---

# Plan execution conventions

Use the planning README as the index and source of truth. Resolve an explicit
plan path first, then the target repository and current branch's managed
`ticket-to-plan` output using the installed Markdown artifact discovery policy,
then legacy `docs/planning/`. Ask the user to choose if multiple plans remain.
Do not require the plan's producer skill to be installed.

Read the README before opening the active phase file. A phase is [ ] pending,
[-] active, or [x] complete; prefer the active phase, otherwise the first
pending phase. Within that phase, select unchecked commits in their stated
order. Resolve missing files, conflicting status, unclear scope, or validation
requirements before making implementation changes. If everything is complete,
report that no work remains without inventing another unit.

Implement only the selected scope. Run the unit's validation, or its phase
exit checks when unit checks are unspecified. Mark a unit complete only after
its checks pass; mark a phase complete only after all its units and exit checks
pass. Preserve completed work and accurate progress when a check fails. Amend
a materially incorrect plan before continuing.

Keep ticket and planning Markdown, including progress edits, outside
implementation commits unless the user explicitly requests otherwise.
`next-commit` leaves implementation uncommitted; `next-phase` creates local
implementation commits. Never expand the invoking skill's branch, staging,
commit, or push permissions. Follow explicit repository requirements.

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
