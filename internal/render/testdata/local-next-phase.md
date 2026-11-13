---
name: next-phase
description: Execute every unchecked commit in the active planning phase, validate and locally commit each implementation unit, and update planning progress locally without pushing.
disable-model-invocation: true
---

# Next phase

For optional integrations, use installed skills and their required references;
otherwise silently follow local defaults, repository conventions, and configured
checks. Do not warn about unavailable optional skills or request their
installation. Explicit repository requirements remain binding.

Use `plan-execution-conventions` when available. Otherwise the README indexes
phase files, [ ] means pending, [-] active, and [x] complete; select the active
phase or first pending phase, then its unchecked units in order.

Resolve an explicit plan path first; otherwise find the current branch's managed
`ticket-to-plan` output, falling back to legacy `docs/planning/`. Read only the
planning README first, resolve the active phase, then process its unchecked
units in order. Resolve missing phase files, ambiguous status, scope, or
validation before implementation. If no unchecked work remains, report that
without creating another unit. The plan's producer skill need not be installed.

Guard the branch: ask for a user-provided feature branch when on main or master,
and stop for unrelated worktree changes. Each unit must stay atomic: implement
its stated scope, apply relevant coding standards, run its validation, mark
progress locally, stage only its implementation files and required tests or
configuration, and create the planned local commit. Use each unit's validation,
or its phase exit checks when none is specified. Mark units complete only
after their checks pass; mark the phase complete only after all its units and
exit checks pass.

Do not stage or commit managed output artifacts.

Before editing code for every unit, inspect the planned implementation paths
and apply relevant language standards. Include formatting changes in that unit
and rerun validation before committing. Never push.

If validation or standards checks fail, or work is cancelled, stop with prior
commits and accurate planning state intact. If the plan no longer matches
reality, amend the plan before continuing.

Finish with a phase summary, commit table, remaining work, and the reminder that
commits are local. The natural next step is `review-branch` (optional integration),
followed by `pr-description` (optional integration).

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

### Output location for this skill

Use `<repo-root>/<branch-slug>/<skill-name>/...` unless the user supplies a path.
No centralized output root is configured for this installation.
These rules apply only while performing this skill.
