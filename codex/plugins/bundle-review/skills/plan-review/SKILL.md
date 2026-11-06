---
name: plan-review
description: Review a phase-indexed implementation plan for vague scope, missing validation, hidden dependencies, unsafe cutovers, and missing risks; explicit invocation only.
---

# Plan review

Read the planning README and every listed phase file. Resolve an explicitly
supplied plan path first; otherwise search the current branch's managed
`ticket-to-plan` output directory, requiring the user to choose if multiple
plans exist, then fall back to legacy `docs/planning/`. Do not implement code.

This is a static plan review. Validation commands in the plan describe future
implementation checks; do not execute them to approve the plan. Run a check
only for an explicit user request, a repository rule covering the changed
files, or a concrete uncertainty affecting review. Explain the reason and use
the narrowest relevant check.

Check that the README is the entrypoint, one phase is active or clearly next,
each phase has atomic commits and observable exit checks, dependencies are
explicit, phases remain mergeable, and irreversible or security-sensitive work
has rollback and risk treatment. Report issues only; if none remain, say Ready
to implement.

Check that Context contains background that affects implementation decisions,
without planning history, inspection logs, generic disclaimers, or repeated
goals.

For up to three passes, fix Critical and Medium planning defects in place,
asking before a change would alter intent. Low-risk tightening is optional.
After readiness, point to next-commit or next-phase; do not start either one.


## Markdown artifact outputs

When a skill writes a Markdown artifact and the user has not supplied an
explicit path, use the installed output-location policy. In repo-local mode,
write beneath `<repo-root>/<branch-slug>/<skill-name>/`; in external mode,
write beneath `<configured-output-root>/<repo-slug>/<branch-slug>/<skill-name>/`.
Create slugs by lowercasing the repository directory name or branch name,
replacing each run of characters outside `a-z`, `0-9`, `.`, `_`, and `-` with
`-`, trimming leading and trailing `-` characters, and using `repo` or
`branch` when the result is empty. For detached HEADs, use
`detached-<short-sha>`. Keep all files from one invocation under its skill
directory, avoid overwriting existing artifacts, and report the exact path.
An explicit user-provided path always wins. This policy applies to existing
skills and every future skill that writes Markdown.

## Markdown artifact discovery

When a skill consumes Markdown produced by another skill, resolve artifacts in
this order:

1. An explicit path supplied by the user or referenced by the current task.
2. The current Git repository's managed output directory for the current
   branch: `<repo-root>/<branch-slug>/<producer-skill>/`.
3. The configured centralized output directory for the current repository and
   branch: `<configured-output-root>/<repo-slug>/<branch-slug>/<producer-skill>/`.
4. A documented legacy location such as `docs/planning/`, when supported.

Determine `<repo-root>` with the target repository's Git root, not the current
working directory. Determine the branch from that repository; detached HEADs
use `detached-<short-sha>`. Search only the producer skill's directory, keep
the current repository and branch in scope, and ask the user to choose when
multiple plausible artifacts remain. Do not silently search another repository,
branch, or output root.

Skills that form a workflow must name their expected producer skill and use
this discovery order before falling back to repository-wide searches.

### Native output policy

Honor explicit user guidance configuring an output location, including a centralized output root. Otherwise use `<repo-root>/<branch-slug>/<skill-name>/...`. These task-local defaults do not modify global settings.
