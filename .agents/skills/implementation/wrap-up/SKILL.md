---
name: wrap-up
description: >-
  Inspect a committed branch for merge-blocking issues and prepare a concise
  merge-request description; supports the combined workflow or either output alone.
---

# Wrap up a branch

Work as the final engineering checkpoint before a branch is merged. The normal
result has two parts: an issues-only review of committed changes, followed by a
copy-ready merge-request description. The user may instead request only the
review or only the description.

Recognize these modes:

| Request | Result |
| --- | --- |
| No mode specified | Review first, then write the merge-request description. |
| `review-only` | Review committed changes and omit the merge-request draft. |
| `mr-only` | Write the merge-request draft from supplied context or the diff; do not repeat the review unless asked. |

When another implementation skill hands off a completed phase, retain the
default combined behavior unless the user has already selected a mode.

## Review the committed branch

Do not review the working tree as though it were part of the branch. Resolve
the comparison base before inspecting the diff:

1. Use a base named by the user when one is available.
2. Otherwise prefer the repository's configured remote default branch, then
   `origin/main`, then `origin/master`.
3. If no base can be resolved, show the available branches and ask the user to
   choose one.

Inspect `git status --short`, the merge base, the diff summary, and the full
committed diff. If the worktree is dirty, say so explicitly and continue with
`merge-base..HEAD` only; never fold local staged or unstaged changes into the
review without being asked.

Read the surrounding files touched by the diff, plus any ticket, specification,
or plan the user supplied. Search for existing implementations before calling
something missing or duplicated. If the branch has a relevant planning folder,
use its README and active phase as the review's specification: compare the
change with its scope, exclusions, acceptance evidence, and exit checks. Prefer
an explicit planning path; otherwise locate a plausible feature folder under
`docs/planning/`. If no relevant plan exists, omit this comparison.

Report only actionable concerns. Check, as applicable:

- behavior errors, inconsistent state, races, swallowed failures, and bad data flow;
- missing or unverified acceptance criteria and work outside the stated scope;
- notable duplication or unnecessary abstractions;
- permissions, boundary, compatibility, deployment, or test risks;
- language-specific convention problems visible in the changed code.

For Python diffs, load [python-standards](../../coding/python-standards/SKILL.md)
before reviewing code. Always follow its core and linting guidance, then load
only the references relevant to the changed paths, such as tests, frameworks,
migrations, imports, errors, docstrings, or logging. Report only
violations that are visible in this diff, not hypothetical cleanup.

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
confirmed defect from a test gap or an uncertainty. Mention a test gap only
when the changed behavior lacks meaningful coverage; do not request tests when
the existing coverage is sufficient.

If the user asks to turn a finding into a merge-request comment, write three to
five sentences: lead with the problem, explain the impact, and include a
reproduction only for a UI or runtime issue.

## Prepare the merge-request description

Skip this section for `review-only`. Prefer context already supplied in the
conversation, followed by findings from this review, then the planning
README's goal and active-phase summary. Fill only real gaps. Ask once for a
missing ticket identifier or change type when that information materially
changes the draft; otherwise infer the type from the change.

Choose the shape that matches the work:

- For a bug fix or feature, explain `Problem`, `Cause`, and `Fix`.
- For a refactor, chore, or dependency change, explain `Context`, `Approach`,
  and `Risks`.

Start with one sentence beginning `This MR ...`; add a ticket identifier in
parentheses when known. Keep the description outcome-focused rather than a
file-by-file changelog. Use short bullets or short paragraphs and casual,
high-level language.

For a bug fix or feature:

```markdown
This MR ...

## The problem

- User, client, or deployment impact.

## The cause

- Root cause(s).

## The fix

- Concrete behavior or outcome delivered.
```

For a refactor, chore, or dependency update:

```markdown
This MR ...

## Context

- Why the change is needed and its boundaries.

## Approach

- Main design moves, expected behavior, and verification.

## Risks

- Potential regressions and mitigations, or a brief statement that risk is low.
```

Do not paste raw commit lists unless requested. Add GitLab quick actions only
when the user supplies the people or explicitly asks for them:

```text
/assign @<assignee>
/assign_reviewer @<reviewer>
```

Return the final merge-request description inside a fenced `markdown` block so
it can be copied directly.

## Boundaries and handoff

Do not modify code, create commits, push, or conceal findings by default. If
the user explicitly requests fixes, make only those fixes, apply the routed
standards, run the relevant formatter, linter, and tests, and report what was
verified. Never push.

The usual previous steps are
[next-phase](../next-phase/SKILL.md) or
[next-commit](../next-commit/SKILL.md). After this skill, the user can open or
update the pull request.
