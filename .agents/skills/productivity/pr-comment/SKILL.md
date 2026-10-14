---
name: pr-comment
description: >-
  Assess a pull-request comment and draft a concise, factual, human Markdown
  comment using the supplied comment, code context, findings, or prior output.
  Use when replying to review feedback or reformatting an existing assessment
  into a teammate-facing response.
---

# Create a comment that can be used to highlight issues or respond to comments on PRs

Turn the supplied comment, context, or previous assessment into a short,
copy-ready Markdown reply. Identify the concern, state what the evidence
supports, and give the decision, next step, or one focused question needed to
continue. Do not invent findings, test results, implementation details, or
commitments.

Be factual, helpful, and empathetic. Acknowledge valid feedback plainly,
correct misunderstandings respectfully, and never assign or assume blame. Keep
it concise; use bullets or a small table only when they make the response
clearer. Preserve uncertainty when it affects the answer.

Output only the reply in Markdown to the terminal; do not explain the drafting
process or add a wrapper. After outputting it, invoke `$to-clipboard` so that
the exact same Markdown is also copied to the clipboard.

