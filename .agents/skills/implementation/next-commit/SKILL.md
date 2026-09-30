---
name: next-commit
description: Implement exactly the next unchecked unit in a phase-indexed plan, validate it, and update planning progress without creating a git commit.
---

# Next commit

Use the planning README as the sole entrypoint. Load [plan-conventions](../../scoping/plan-conventions/SKILL.md).

1. Find the active phase ([-], otherwise first [ ]) and open its phase file.
2. Check the current branch without switching; pause on main or master.
3. Implement only the first unchecked commit, with no scope expansion.
4. Before final validation, inspect the implementation paths and apply every
   relevant coding-standards pack. For Python changes, load
   [python-standards](../../coding/python-standards/SKILL.md), including its required
   references, and run the repository formatter, linter, and other checks it
   calls for. Include any formatting changes in the unit and rerun validation.
5. Run that commit's validation, or the phase exit checks if none is defined.
6. Mark the commit complete and update the phase status in the README.
7. Report changes, a suggested message, changed files, and the next action. Do
   not run git commit, push, or continue to another unit.

If the plan is materially wrong, stop and propose a docs-first amendment. When a
phase completes, point to [wrap-up](../wrap-up/SKILL.md); otherwise point to [next-phase](../next-phase/SKILL.md).
