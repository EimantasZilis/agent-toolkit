---
name: plan-conventions
description: Shared rules for phase-indexed planning, validation, progress markers, and single-unit or full-phase execution.
---

# Plan conventions

The feature README is the index and source of truth. Unless the user supplies
an explicit path, planning lives under a date-slugged directory beneath the
installed Markdown output policy's `ticket-to-plan` skill directory, with a
README and one file per phase. Legacy plans under `docs/planning/` remain valid
when explicitly supplied or when no managed plan can be found. The README
contains Goal, Context, Out of scope, Assumptions, Phases, and Risks; workflow
instructions live in the skills.

A phase is [ ] pending, [-] active, or [x] complete. A commit is atomic,
predefined, and leaves the product working. New commit entries include concrete
validation commands or manual checks. Keep ticket and planning Markdown,
including progress edits, outside implementation commits unless the user
explicitly asks to include it. Full-phase mode creates local implementation
commits; single-unit mode leaves the git commit to the user.

Use the backlog-ticket structure in [requirements-to-tickets](../requirements-to-tickets/SKILL.md). For auth,
uploads, personal data, or new external boundaries, route through
[security-check](../../security/security-check/SKILL.md) or [security-review](../../security/security-review/SKILL.md).
