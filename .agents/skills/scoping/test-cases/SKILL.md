---
name: test-cases
description: Create prioritized Markdown manual test cases for a branch's changes using the ticket, specification, plans, diff, tests, and relevant repository or business evidence.
---

# Manual test cases

Produce a practical manual test-case document for validating the observable
behavior of changes on the current branch. This skill is for manual application
testing, not implementation, automated test authoring, or changing a ticket or
business-system record.

## Establish the change set

1. Resolve the comparison base in this order: a base named by the user, the
   repository's configured remote default branch, `origin/main`, then
   `origin/master`. If no base can be resolved, ask the user to choose one
   rather than guessing.
2. Inspect the complete change set from the base to the current working state,
   including committed, staged, and unstaged changes. Report a dirty worktree
   when it affects scope.
3. Identify the ticket, specification, requirements, or implementation plan
   from the request, local files, branch name, or connected business source.
   If no source is available, continue from repository evidence and record the
   limitation.
4. Inspect the ticket or specification and relevant plan/context artifacts
   before the diff. Then inspect the diff, affected application paths,
   existing automated tests, documentation, configuration, schema, fixtures,
   and rollout context when they clarify behavior.
5. Use automated tests to understand what is already covered. Do not turn unit
   or integration assertions into manual cases. Manual cases must exercise an
   observable application flow through the UI, API/backend, job, integration,
   or a combination of these.

Prefer connected business sources for intent, acceptance criteria, priority,
terminology, and rollout constraints. Prefer current repository behavior for
what can actually be tested. Read business systems only when access is
available and keep that access read-only.

## Select and design cases

Trace each changed user-visible or externally observable flow end to end:
setup, actor and permissions, feature flags, input data, trigger, state
changes, outputs, and cleanup. Keep the plan tied to evidence from the branch.
Do not present an unverified behavior as an acceptance criterion.

Start with happy paths, then include only the edge cases most likely to expose
a meaningful regression or harmful state change. Include an edge case when it
tests a changed boundary, validation or failure mode, permission boundary,
lifecycle transition, compatibility concern, repeated execution, or data
integrity risk. Omit exhaustive permutations and cases that only repeat
automated assertions.

Order cases by execution priority:

- **P0 — release blocker:** core outcome, data integrity, access control,
  migration or compatibility, or a failure that makes the feature unusable.
- **P1 — high:** important alternate flows, validation, recovery, and likely
  regression paths.
- **P2 — useful coverage:** lower-risk variants or boundary cases to run when
  time and environment permit.

Every case must be executable without interpreting hidden assumptions. Include
concrete prerequisites, role or permission context, data and starting state,
actions with concrete inputs, expected results after meaningful transitions,
and cleanup, reset, or evidence capture when relevant. Split cases when setup,
risk, or expected outcome changes materially. Combine closely related variants
when the setup and expected behavior are the same.

Choose the environment from evidence in the request, ticket, plan, rollout
configuration, or repository. Do not invent a staging or production target.
Mark missing or conflicting environment information as `Needs confirmation`,
and do not imply that a production test is safe merely because it is listed.

Mark uncertainty explicitly as `Unknown`, `Assumption`, or `Needs
confirmation`. Call out conflicts between the ticket, plan, code, and tests.

## Required Markdown output

Write a Markdown file. If no output path is requested, use
`docs/test-plans/<yyyy-mm-dd>-<ticket-or-branch-slug>.md`, creating the
directory as needed. Do not overwrite an existing plan without explicit
direction.

Use this structure, adapting or omitting only sections that are genuinely
irrelevant:

```markdown
# Manual test cases: <ticket or change title>

> Status: draft | Branch: `<branch>` | Ticket: `<reference>` | Evidence inspected: <short scope and date>

## Purpose and coverage summary

<What changed, what this document proves, and the highest-risk areas.>

## Scope and evidence

| Source | What it contributed | Limitation or confidence |
| --- | --- | --- |
| Ticket / issue | Intent and acceptance criteria | Confirmed / unavailable / ... |
| Plan or context | Intended change surface | ... |
| Branch diff and automated tests | Implemented behavior and existing coverage | ... |

## Test environment and data

- Branch/build/feature flags: ...
- Environment(s): local, staging, production, or `Needs confirmation`
- Required services, accounts, roles, and permissions: ...
- Test data and starting state: ...
- Evidence to retain: ...

## Priority test cases

| Test # | Priority | Environment | Test title | What are we testing? | Why are we testing it? | Steps/actions | Expected results |
| --- | --- | --- | --- | --- | --- | --- | --- |
| P0-01 | P0 | ... | ... | ... | ... | 1. ...<br>2. ... | ... |

**Cleanup/evidence notes:** <case-specific reset, cleanup, or evidence instructions.>

## Traceability and gaps

| Ticket criterion or changed flow | Test case(s) | Coverage / gap |
| --- | --- | --- |
| ... | ... | Covered / partial / not manually testable |

- **Unknown:** ...
- **Assumption:** ...
- **Needs confirmation:** ...

## Execution notes

<Ordering, stop conditions, known non-testable areas, and how to report a failure.>
```

Keep table cells concise and use `<br>` for multiple steps. If a case needs
substantial setup, cleanup, or evidence instructions, put those details in a
short case-specific subsection immediately below the table and link it from
the relevant row. Do not sacrifice reproducibility for compactness.

Use repository-relative Markdown links with line numbers where useful. The
final response should report the written path, sources inspected, and material
gaps. Do not claim that cases passed unless they were actually executed.
