---
name: ticket-to-plan
description: Convert a ticket from a file, chat, or Jira body into an implementation-ready planning folder; do not implement application code.
---

# Ticket roadmap

A ticket is required. Accept a local ticket, pasted body, or chat task; if none
exists, route to [requirements-to-tickets](../requirements-to-tickets/SKILL.md).
Investigation-only work routes to [quick-plan](../quick-plan/SKILL.md).

Normalize title, goal, context, exclusions, assumptions, acceptance criteria,
risks, and suggested commits. Preserve suggested order where useful, expand each
unit with concrete scope and validation, and default to one phase unless
independent mergeable boundaries justify more. In hotfix mode use one phase and
at most two commits.

Keep the resulting plan sized for one focused, self-contained PR. Each phase
must deliver one coherent outcome and include only the smallest necessary
implementation, tests, configuration, and documentation. Split a phase when
it combines multiple user outcomes, independently deployable or revertible
changes, separate ownership, an optional enhancement, or a prerequisite that
can merge independently. Do not use phases to group unrelated cleanup or
future work, and do not let a phase depend on undocumented work hidden in
another phase. During planning, perform a PR-size check: if the scope would
produce a broad or difficult-to-review diff, narrow the phase or create
explicitly dependent plans/phases with clear handoffs and acceptance criteria.
Keep each phase's commits focused on that phase's outcome.

After user confirmation, write README.md and phase-N.md under
docs/planning/date-slug/; for chat-only input also retain a retrospective
ticket under docs/jira/. Run plan-review as the checkpoint. Stop at
ready-to-implement; never write application code. Run
[plan-review](../plan-review/SKILL.md) as the checkpoint.

Start each generated `README.md` with a concise Mermaid diagram that gives the
overview of the work and shows phase dependencies before the detailed Goal,
Context, Phases, and Risks sections. Keep node and edge descriptions brief but
understandable, and keep the diagram synchronized with the phase order and
dependency statements.
