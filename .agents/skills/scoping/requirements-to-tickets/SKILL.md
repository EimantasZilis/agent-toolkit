---
name: requirements-to-tickets
description: Break a feature or epic brief into self-contained Markdown tickets; do not write application code.
---

# Requirements to tickets

If the request is investigation-only, route to [quick-plan](../quick-plan/SKILL.md). Otherwise capture
the outcome, users, constraints, and completion evidence, then produce tickets
that can stand alone. Use the ticket structure below.

For engineer tickets include lightweight suggested commits expressing what and
why; leave where, how, and detailed validation to [ticket-to-plan](../ticket-to-plan/SKILL.md). For product
manager mode omit commits. Review the tickets for missing boundaries,
dependencies, ambiguity, and observable acceptance criteria before presenting
them.

When the requested deliverable includes a `README.md`, open it with a concise
Mermaid diagram that shows the work and ticket dependencies, then explain each
ticket in detail below. Use short, understandable node labels and keep the
diagram synchronized with the ticket order and dependency statements.

Keep each ticket small enough for one self-contained, independently reviewable
PR. A ticket should have one primary outcome, one clear completion boundary,
and no unrelated cleanup or follow-up work. Split work when it contains
multiple user outcomes, independently deployable or revertible changes,
separate ownership, a prerequisite that could land on its own, or a review
that would be difficult to understand as one diff. Prefer vertical slices
that include the smallest required code, tests, configuration, and docs for
their outcome. Make dependencies explicit instead of hiding several large
changes in one ticket. Before presenting tickets, check that every ticket can
be described in a short title and that its suggested commits would form a
focused PR; if not, decompose it further.

Do not create files unless asked; when asked, write only ticket files.

## Ticket structure

Use Markdown that can be copied into the team's issue tracker without requiring
tracker-specific syntax.

```markdown
## Outcome

**As a** <role>,
**I want to** <high_level_objective>
**So that** <high_level_outcome>

## Context
Add a few sentences what this ticket is for and why it is needed in the wider context.

## Details
Cover what to build. Use numbered lists or bullet points for specifics. Cover:

- Included work
- Important constraints
- Explicit exclusions

## Acceptance criteria
Cover what needs to happen for the ticket to be "complete". 
Be specific. Don't add generic "tests must pass" lines or similar.
Acceptance criteria could cover:
- Observable behavior
- Edge cases
- Required validation

## Delivery notes
Optional section to add 
- Dependencies
- Suggested implementation units
- Risks or rollout concerns
```

Keep each section short and concrete. Omit sections that have no meaningful
content. For engineer tickets, put suggested commits under Delivery notes and
state what each commit accomplishes; leave file-level implementation detail to
[ticket-to-plan](../ticket-to-plan/SKILL.md). Include tests or manual checks in
Completion checks, not as separate tickets. Avoid copied framework examples,
tracker-specific field names, and sections that merely restate the outcome.
