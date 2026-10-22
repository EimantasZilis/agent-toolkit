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
phase completes, point to [review-branch](../../review/review-branch/SKILL.md)
and [pr-description](../../productivity/pr-description/SKILL.md); otherwise
point to [next-phase](../next-phase/SKILL.md).
