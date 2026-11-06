---
name: quick-plan
description: Run a bounded feasibility investigation and give one recommendation without implementing or creating a commit plan.
disable-model-invocation: true
---

# Quick plan

Use for should-we, feasibility, technology choice, or unknowns blocking a build
decision. Define the question and exclusions, list only decision-blocking
unknowns, inspect the smallest relevant evidence, and record assumptions.

Report findings as evidence tied to those questions, then choose exactly one:
proceed, requirements-to-tickets, or abandon. Explain the choice in one
sentence and a few reasons. A build-worthy result can chain to
`requirements-to-tickets` (optional integration); a ticketed build
can then use `ticket-to-plan` (optional integration). Do not
implement code or invent a phase plan here.
