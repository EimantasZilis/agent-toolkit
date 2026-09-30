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
are involved, say that the short path is inconclusive and continue with
 [security-review](../security-review/SKILL.md) in the same session. For changes involving identity, uploads,
personal data, or new external boundaries, preserve the security gate in the
resulting plan.
