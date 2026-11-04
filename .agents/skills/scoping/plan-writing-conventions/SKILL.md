---
name: plan-writing-conventions
description: Define the structure, scope, validation, and risk treatment of phase-indexed implementation plans when creating or revising them.
---

# Plan writing conventions

The feature README is the index and source of truth. Unless the user supplies
an explicit path, planning lives under a date-slugged directory beneath the
installed Markdown output policy's `ticket-to-plan` skill directory, with a
README and one file per phase. Legacy plans under `docs/planning/` remain valid
when explicitly supplied or when no managed plan can be found. The README
contains Goal, Context, Out of scope, Assumptions, Phases, and Risks; workflow
instructions live in the skills.

Context explains existing behavior, the problem, and constraints that affect
implementation choices. Include relevant code paths or dependencies when they
help the implementer. Exclude planning history, inspection logs, generic
disclaimers, and restatements of Goal. Keep this required section brief when
little background is needed.

A phase is [ ] pending, [-] active, or [x] complete. Make the next phase
unambiguous. Each phase delivers one coherent, reviewable outcome; each commit
is atomic, predefined, and leaves the product working. Include concrete scope,
acceptance criteria, and validation commands or manual checks for every unit,
plus observable phase exit checks. State dependencies and exclusions explicitly.

Validation commands in a plan are future implementation requirements, not
instructions to run them while authoring. Default to checking the plan's
accuracy, links, structure, and dependencies. Run tests or builds only for an
explicit user request, a repository rule covering the changed files, or a
concrete uncertainty that affects the plan; explain the reason and choose the
narrowest check. Distinguish performed checks from proposed validation.

Record assumptions, compatibility constraints, rollback needs, and unresolved
risks. For identity, uploads, personal data, or new external boundaries, record
any security assessment needed before implementation and its acceptance evidence;
do not automatically invoke another skill. Do not include execution procedures
in the generated plan or create implementation commits while authoring it.
