---
name: next-phase
description: Execute every unchecked commit in the active planning phase, validate and locally commit each implementation unit, and update planning progress locally without pushing.
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
commits are local. The natural next step is [review-branch](../../review/review-branch/SKILL.md),
followed by [pr-description](../../productivity/pr-description/SKILL.md).
