---
name: pr-description
description: >-
  Prepare a concise, copy-ready pull-request or merge-request description from
  supplied context, a branch diff, findings, or a plan.
---

# Prepare a pull-request description

Use context already supplied in the conversation, followed by findings from a
branch review, then the planning README's goal and active-phase summary. Fill
only real gaps. Ask once for a missing ticket identifier or change type when
that information materially changes the draft; otherwise infer the type from
the change.

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


Return the final pull-request description inside a fenced `markdown` block so
it can be copied directly. Do not imply that a branch review, tests, linters,
or builds were performed unless the supplied context provides that evidence.
After returning it, invoke `$to-clipboard` so the same Markdown remains visible
in the terminal and is copied to the clipboard.
