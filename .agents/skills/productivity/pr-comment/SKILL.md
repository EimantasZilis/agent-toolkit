---
name: pr-comment
description: >-
  Assess a pull-request comment and draft a concise, factual, human Markdown
  response using the supplied comment, code context, findings, or prior output.
  Use when replying to review feedback or reformatting an existing assessment
  into a teammate-facing response.
---

# Respond to a pull-request comment

Turn the supplied comment, context, or previous assessment into a short,
copy-ready Markdown reply. Identify the concern, state what the evidence
supports, and give the decision, next step, or one focused question needed to
continue. Do not invent findings, test results, implementation details, or
commitments.

Be factual, helpful, and empathetic. Acknowledge valid feedback plainly,
correct misunderstandings respectfully, and never assign or assume blame. Keep
it concise; use bullets or a small table only when they make the response
clearer. Preserve uncertainty when it affects the answer.

Return only the reply in Markdown, without explaining the drafting process.
After returning it, invoke `$to-clipboard` so the same Markdown remains visible
in the terminal and is copied to the clipboard.

