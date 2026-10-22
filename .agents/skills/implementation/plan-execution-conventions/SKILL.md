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
