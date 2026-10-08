---
name: test-plan
description: Create a prioritized Markdown manual test plan for a branch's ticketed changes using the ticket, implementation plans, diff, tests, and relevant repository or business evidence.
---

# Ticket test plan

Produce a practical manual test plan for validating the changes on the current
branch. This skill is for test planning, not implementation, automated test
authoring, or changing a ticket or business-system record.

## Evidence and scope

1. Identify the ticket reference from the request, a local ticket file, a
   pasted body, a Jira or issue URL, or the branch name. If more than one
   ticket is plausible, state the assumption and continue with the smallest
   defensible scope.
2. Inspect the implementation plan and any ticket context or requirements
   artifacts before reading the diff. Then inspect the branch diff against its
   likely base, affected tests, relevant documentation, configuration, schema,
   fixtures, and recent history when those clarify behavior.
3. Read connected business sources when access is available. Use them for
   intent, acceptance criteria, priority, terminology, and rollout constraints;
   use the repository and branch for the behavior that can actually be tested.
   If a source is unavailable, record that limitation rather than implying it
   was checked.
4. Trace the changed user-visible or externally observable flows end to end.
   Include setup, actor or permissions, input data, trigger, expected result,
   and cleanup or reset needs. Prefer evidence over names or assumptions.
5. Cover the primary happy paths first, then the few edge cases most likely to
   expose regressions or harmful data/state changes. Use judgment: include an
   edge case when it tests a changed boundary, failure mode, permission or
   lifecycle transition, compatibility concern, or meaningful data constraint;
   omit exhaustive permutations and trivial variants.

Keep the plan tied to the branch. Do not present an unverified behavior as an
acceptance criterion. Mark uncertainty as `Unknown`, `Assumption`, or `Needs
confirmation`, and call out any ticket/plan/code conflict.

## Priority and test-case design

Order test cases by execution priority:

- **P0 — release blocker:** the core outcome, data integrity, access control,
  migration/compatibility, or a failure that makes the feature unusable.
- **P1 — high:** important alternate flows, validation, recovery, and the most
  likely regression paths.
- **P2 — useful coverage:** lower-risk variants or boundary cases that should be
  run when time and environment permit.

Give every test case a stable identifier and a short title. Each case should
contain:

- why it matters and the risk it covers;
- prerequisites, including role, feature flag, data, environment, and starting
  state;
- numbered action steps with concrete inputs;
- expected results after each meaningful transition or at the end of the flow;
- cleanup, reset, or evidence to capture when relevant.

Use one case for a coherent behavior. Split a case when its setup, risk, or
expected outcome changes materially. Combine closely related permutations when
that keeps the plan readable and the expected behavior identical.

## Required Markdown output

Write a Markdown file. If no output path is requested, use
`docs/test-plans/<yyyy-mm-dd>-<ticket-or-branch-slug>.md`, creating the
directory as needed. Do not overwrite an existing plan without explicit
direction.

Use this structure, adapting or omitting only sections that are genuinely
irrelevant:

```markdown
# Manual test plan: <ticket or change title>

> Status: draft | Branch: `<branch>` | Ticket: `<reference>` | Evidence inspected: <short scope and date>

## Purpose and coverage summary

<What changed, what this plan proves, and the highest-risk areas.>

## Scope and evidence

| Source | What it contributed | Limitation or confidence |
| --- | --- | --- |
| Ticket / issue | Intent and acceptance criteria | Confirmed / unavailable / ... |
| Plan or context | Intended change surface | ... |
| Branch diff and tests | Implemented behavior and regression clues | ... |

## Test environment and data

- Branch/build/feature flags: ...
- Required services, accounts, roles, and permissions: ...
- Test data and starting state: ...
- Evidence to retain: ...

## Priority test cases

### P0-01 — <short title>

**Risk covered:** <why this case is high priority>

**Prerequisites:** <role, data, flags, state>

**Steps and expected results:**

1. <Action> — <Expected result>
2. <Action> — <Expected result>

**Cleanup/evidence:** <reset or capture, if needed>

### P1-01 — <short title>

...

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

Use repository-relative Markdown links with line numbers where useful. Keep
the document concise enough to execute, but do not remove setup or expected
results that make a case reproducible. The final response should report the
written path, sources inspected, and any material gaps; do not claim the
branch passed unless the cases were actually executed.
