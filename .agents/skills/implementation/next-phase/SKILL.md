---
name: next-phase
description: Execute every unchecked commit in the active planning phase, validate and locally commit each unit with its progress docs, then stop without pushing.
---

# Next phase

Load [plan-conventions](../../scoping/plan-conventions/SKILL.md). Read only the planning README first, resolve the active
phase, then process its unchecked units in order.

Guard the branch: ask for a user-provided feature branch when on main or master,
and stop for unrelated worktree changes. Each unit must stay atomic: implement
its stated scope, apply relevant coding standards, run its validation, mark
progress, stage only its implementation and progress files, and create the
planned local commit. Before validation and staging for every unit, inspect the
changed implementation paths and apply every relevant coding-standards pack.
For Python changes, load [python-standards](../../coding/python-standards/SKILL.md),
including its required references, and run the repository formatter, linter,
and other checks it calls for. Include any formatting changes in that unit and
rerun validation before committing. Never push. If validation or standards
checks fail, stop with prior commits and planning state intact. If the plan no
longer matches reality, amend the plan before continuing.

Finish with a phase summary, commit table, remaining work, and the reminder that
commits are local. The natural next step is [wrap-up](../wrap-up/SKILL.md).
