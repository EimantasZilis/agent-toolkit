---
name: plan-conventions
description: Shared rules for phase-indexed planning, validation, progress markers, and single-unit or full-phase execution.
---

# Plan conventions

The feature README is the index and source of truth. Planning lives under
docs/planning/yyyy-mm-dd-feature/ with a README and one file per phase. The
README contains Goal, Context, Out of scope, Assumptions, Phases, and Risks;
workflow instructions live in the skills.

A phase is [ ] pending, [-] active, or [x] complete. A commit is atomic,
predefined, and leaves the product working. New commit entries include concrete
validation commands or manual checks. Progress edits travel with the
implementation commit in full-phase mode; single-unit mode leaves the git
commit to the user.

Use the backlog-ticket structure in [requirements-to-tickets](../requirements-to-tickets/SKILL.md). For auth,
uploads, personal data, or new external boundaries, route through
[security-check](../../security/security-check/SKILL.md) or [security-review](../../security/security-review/SKILL.md).
