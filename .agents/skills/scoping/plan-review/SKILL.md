---
name: plan-review
description: Review a phase-indexed implementation plan for vague scope, missing validation, hidden dependencies, unsafe cutovers, and missing risks; explicit invocation only.
---

# Plan review

Read the planning README and every listed phase file. Do not implement code.

Check that the README is the entrypoint, one phase is active or clearly next,
each phase has atomic commits and observable exit checks, dependencies are
explicit, phases remain mergeable, and irreversible or security-sensitive work
has rollback and risk treatment. Report issues only; if none remain, say Ready
to implement.

For up to three passes, fix Critical and Medium planning defects in place,
asking before a change would alter intent. Low-risk tightening is optional.
After readiness, point to next-commit or next-phase; do not start either one.

