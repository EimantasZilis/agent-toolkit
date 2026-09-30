---
name: security-review
description: Perform a detailed, pragmatic security assessment across a system or repositories, including exploitability, controls, remediation, impact, and confidence.
---

# Security review

Use this for architectural, multi-repository, or ambiguous findings. For one
clear alert with a short action, use [security-check](../security-check/SKILL.md).

Collect only relevant code, configuration, deployment, identity, data-flow, and
dependency evidence. Model realistic attackers and trust boundaries; distinguish
a theoretical weakness from a reachable exploit. Cover existing controls,
affected systems, remediation alternatives, compatibility or customer impact,
hidden risks, and confidence with explicit unknowns.

Return TL;DR, classification, exploitability, controls, affected systems,
remediation options, recommendation, impact, additional considerations, and
confidence. Missing context lowers confidence; do not invent a vulnerability.
Feed implementation work into [requirements-to-tickets](../../scoping/requirements-to-tickets/SKILL.md)
or [ticket-to-plan](../../scoping/ticket-to-plan/SKILL.md).
