---
name: requirements-to-tickets
description: Break a feature or epic brief into self-contained Markdown tickets; do not write application code.
disable-model-invocation: true
---

# Requirements to tickets

If the request is investigation-only, route to `quick-plan` (optional integration). Otherwise capture
the outcome, users, constraints, and completion evidence, then produce tickets
that can stand alone. Use the ticket structure below.

For engineer tickets include lightweight suggested commits expressing what and
why; leave where, how, and detailed validation to `ticket-to-plan` (optional integration). For product
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

Do not create files unless asked; when asked, write only ticket files. If no
path is supplied, place them beneath the installed Markdown output policy's
`requirements-to-tickets` skill directory. An explicit output path wins.

## Ticket structure

Use Markdown that can be copied into the team's issue tracker without requiring
tracker-specific syntax.

```markdown
## Outcome

**As a** <role>,
**I want to** <high_level_objective>
**So that** <high_level_outcome>

## Context
Explain current behavior, the problem, and constraints that affect this work.
Include only background needed to understand the ticket. Omit planning history,
inspection logs, generic disclaimers, and restatements of Outcome.

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
`ticket-to-plan` (optional integration). Include tests or manual checks in
Completion checks, not as separate tickets. Avoid copied framework examples,
tracker-specific field names, and sections that merely restate the outcome.

## Markdown artifact outputs

When a skill writes a Markdown artifact and the user has not supplied an
explicit path, use the installed output-location policy. In repo-local mode,
write beneath `<repo-root>/<branch-slug>/<skill-name>/`; in external mode,
write beneath `<configured-output-root>/<repo-slug>/<branch-slug>/<skill-name>/`.
Create slugs by lowercasing the repository directory name or branch name,
replacing each run of characters outside `a-z`, `0-9`, `.`, `_`, and `-` with
`-`, trimming leading and trailing `-` characters, and using `repo` or
`branch` when the result is empty. For detached HEADs, use
`detached-<short-sha>`. Keep all files from one invocation under its skill
directory, avoid overwriting existing artifacts, and report the exact path.
An explicit user-provided path always wins. This policy applies to existing
skills and every future skill that writes Markdown.

## Markdown artifact discovery

When a skill consumes Markdown produced by another skill, resolve artifacts in
this order:

1. An explicit path supplied by the user or referenced by the current task.
2. The current Git repository's managed output directory for the current
   branch: `<repo-root>/<branch-slug>/<producer-skill>/`.
3. The configured centralized output directory for the current repository and
   branch: `<configured-output-root>/<repo-slug>/<branch-slug>/<producer-skill>/`.
4. A documented legacy location such as `docs/planning/`, when supported.

Determine `<repo-root>` with the target repository's Git root, not the current
working directory. Determine the branch from that repository; detached HEADs
use `detached-<short-sha>`. Search only the producer skill's directory, keep
the current repository and branch in scope, and ask the user to choose when
multiple plausible artifacts remain. Do not silently search another repository,
branch, or output root.

Skills that form a workflow must name their expected producer skill and use
this discovery order before falling back to repository-wide searches.

### Native output policy

Honor explicit user guidance configuring an output location, including a centralized output root. Otherwise use `<repo-root>/<branch-slug>/<skill-name>/...`. These task-local defaults do not modify global settings.
