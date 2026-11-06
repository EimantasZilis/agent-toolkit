---
name: plan-writing-conventions
description: Define the structure, scope, validation, and risk treatment of phase-indexed implementation plans when creating or revising them.
---

# Plan writing conventions

The feature README is the index and source of truth. Unless the user supplies
an explicit path, planning lives under a date-slugged directory beneath the
installed Markdown output policy's `ticket-to-plan` skill directory, with a
README and one file per phase. Legacy plans under `docs/planning/` remain valid
when explicitly supplied or when no managed plan can be found. The README
contains Goal, Context, Out of scope, Assumptions, Phases, and Risks; workflow
instructions live in the skills.

Context explains existing behavior, the problem, and constraints that affect
implementation choices. Include relevant code paths or dependencies when they
help the implementer. Exclude planning history, inspection logs, generic
disclaimers, and restatements of Goal. Keep this required section brief when
little background is needed.

A phase is [ ] pending, [-] active, or [x] complete. Make the next phase
unambiguous. Each phase delivers one coherent, reviewable outcome; each commit
is atomic, predefined, and leaves the product working. Include concrete scope,
acceptance criteria, and validation commands or manual checks for every unit,
plus observable phase exit checks. State dependencies and exclusions explicitly.

Validation commands in a plan are future implementation requirements, not
instructions to run them while authoring. Default to checking the plan's
accuracy, links, structure, and dependencies. Run tests or builds only for an
explicit user request, a repository rule covering the changed files, or a
concrete uncertainty that affects the plan; explain the reason and choose the
narrowest check. Distinguish performed checks from proposed validation.

Record assumptions, compatibility constraints, rollback needs, and unresolved
risks. For identity, uploads, personal data, or new external boundaries, record
any security assessment needed before implementation and its acceptance evidence;
do not automatically invoke another skill. Do not include execution procedures
in the generated plan or create implementation commits while authoring it.

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
