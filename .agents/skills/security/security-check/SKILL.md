---
name: security-check
description: Give a short fix, accept, or monitor decision for a focused security alert, finding, or question.
---

# Security check

Use a focused pass for one CVE, dependency alert, pull-request concern, or
is-this-bad question. State the finding, realistic exposure, the recommended
action (fix, accept, or monitor), and the one or two facts that drive it. Do
not fabricate severity or exploitability.

If the evidence is ambiguous, the scope is architectural, or multiple systems
are involved, say that the focused assessment is inconclusive, explain which
questions require a broader assessment, and recommend `security-review`.
Stop the focused assessment there: do not automatically invoke or install
another skill, perform the deeper investigation locally, or give an unsupported
fix, accept, or monitor verdict. This recommendation concerns the scope of the
assessment, not whether another skill is installed. For changes involving
identity, uploads, personal data, or new external boundaries, preserve the
security gate in the resulting plan.
