---
name: review-branch
description: >-
  Inspect a committed branch for merge-blocking issues and report actionable,
  evidence-based findings before merge.
---

# Review a branch

Work as the final engineering checkpoint before a branch is merged. Produce
an issues-only review of committed changes.

Resolve the comparison base before inspecting the diff:

1. Use a base named by the user when one is available.
2. Otherwise prefer the repository's configured remote default branch, then
   `origin/main`, then `origin/master`.
3. If no base can be resolved, show the available branches and ask the user to
   choose one.

Inspect `git status --short`, the merge base, the diff summary, and the full
committed diff. If the worktree is dirty, say so explicitly and continue with
`merge-base..HEAD` only; never fold local staged or unstaged changes into the
review without being asked.

Read the surrounding files touched by the diff, plus any ticket,
specification, or plan the user supplied. Search for existing implementations
before calling something missing or duplicated. If the branch has a relevant
planning folder, use its README and active phase as the review's specification:
compare the change with its scope, exclusions, acceptance evidence, and exit
checks. Prefer an explicit planning path; otherwise locate the current
branch's managed `ticket-to-plan` output and then a plausible feature folder
under `docs/planning/`. If no relevant plan exists, omit this comparison.

Use a multi-pass review. Keep the working analysis structured, but preserve
the issues-only output contract below:

1. **Scope and contract pass.** Map every changed file to its callers,
   persistence, configuration, deployment path, and tests. Extract intended
   behavior, invariants, compatibility expectations, and exclusions from the
   diff and available specifications. Do not treat an absent specification as
   permission to invent requirements.
2. **Behavior and edge-case pass.** For each changed behavior, maintain a
   small matrix of relevant scenarios and mark each as covered, unverified,
   contradicted, or not applicable. Consider dimensions the change actually
   exposes: empty or missing input, malformed or boundary values, duplicates
   and repeated execution, retries and partial failure, timeout or dependency
   failure, stale or concurrent state, permission boundaries, legacy data or
   schema versions, and configuration or deployment transitions. For every
   conditional or external call, try the counterexample: what if the
   condition is false, the call fails, the result is delayed, or the operation
   runs twice?
3. **Data-flow and invariant pass.** Trace inputs to side effects and outputs.
   Check validation and authorization order, state transitions, transaction
   boundaries, cleanup, exception propagation, resource limits, and
   observability. Compare changed behavior with existing implementations and
   important callers to find incompatible assumptions or duplicated logic.
4. **Evidence pass.** Inspect relevant tests, fixtures, CI configuration, and
   planning exit checks, then map meaningful evidence to the matrix. This
   review is static: do not claim that tests, linters, or builds passed unless
   the user explicitly requests execution and the command was actually run.

Do not conclude until every relevant changed behavior has a matrix status and
high-risk paths have received the deeper passes. Escalate review depth for
authentication or authorization, data deletion or migration, concurrency,
public interfaces, billing, secrets, or deployment changes. Keep unresolved
uncertainty explicit rather than converting it into a hypothetical defect.

Report only actionable concerns. Check, as applicable:

- behavior errors, inconsistent state, races, swallowed failures, and bad data flow;
- missing or unverified acceptance criteria and work outside the stated scope;
- notable duplication or unnecessary abstractions;
- permissions, boundary, compatibility, deployment, or test risks;
- language-specific convention problems visible in the changed code.

For Python diffs, load [python-standards](../python-standards/SKILL.md)
before reviewing code. Always follow its core and linting guidance, then load
only the references relevant to the changed paths, such as tests, frameworks,
migrations, imports, errors, docstrings, or logging. Report only violations
that are visible in this diff, not hypothetical cleanup.

## Findings format

Keep the review issues-only: do not add praise or a general change summary.
If there are no actionable concerns, write exactly: `No issues found.`

When there are findings, use one Markdown table for defects and plan/spec gaps:

```markdown
| Severity | Finding | Location | Why it matters | Recommended action |
| --- | --- | --- | --- | --- |
| Critical — fix before merge | Short title | `path/to/file` | Impact and evidence | Specific correction or mitigation |
| Medium — fix before merge | Short title | `path/to/file` | Impact and evidence | Specific correction or mitigation |
| Low — follow-up ticket | Short title | `path/to/file` | Impact and evidence | Concrete follow-up |
| Low (spec) — acceptable with note | Short title | `docs/planning/...` | Missing or unclear requirement | Confirmation or documentation |
```

Order rows from highest to lowest severity. Use `-` for the location only when
the concern genuinely has no primary file. Explain unfamiliar code in plain
language, include file and line references when available, and distinguish a
confirmed defect from a test gap or an uncertainty. Every finding must state
the triggering scenario or precondition, cite concrete evidence from the diff
or surrounding code, and make its confidence clear through that distinction.
Mention a test gap only when the changed behavior lacks meaningful coverage;
do not request tests when the existing coverage is sufficient. Deduplicate
findings that share the same root cause.

If the user asks to turn a finding into a merge-request comment, write three to
five sentences: lead with the problem, explain the impact, and include a
reproduction only for a UI or runtime issue.

## Boundaries

Do not modify code, create commits, push, or conceal findings by default. If
the user explicitly requests fixes, make only those fixes, apply the routed
standards, run the relevant formatter, linter, and tests, and report what was
verified. Never push.
