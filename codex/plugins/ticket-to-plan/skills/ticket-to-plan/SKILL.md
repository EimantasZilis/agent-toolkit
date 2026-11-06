---
name: ticket-to-plan
description: Convert a ticket from a file, chat, or Jira body into an implementation-ready planning folder; do not implement application code.
---

# Ticket roadmap

For optional integrations, use installed skills and their required references;
otherwise silently follow local defaults, repository conventions, and configured
checks. Do not warn about unavailable optional skills or request their
installation. Explicit repository requirements remain binding.

A ticket is required. Accept a local ticket, pasted body, or chat task; if none
exists, route to `requirements-to-tickets` (optional integration).
Investigation-only work routes to `quick-plan` (optional integration).

Normalize title, goal, context, exclusions, assumptions, acceptance criteria,
risks, and suggested commits. Preserve suggested order where useful, expand each
unit with concrete scope and validation, and default to one phase unless
independent mergeable boundaries justify more. In hotfix mode use one phase and
at most two commits.

Use `plan-writing-conventions` when available. Otherwise use a README index with
Goal, Context, Out of scope, Assumptions, Phases, and Risks, plus one file per phase
with scoped atomic commits, acceptance criteria, concrete validation, and
phase exit checks. Use [ ] for pending, [-] for active, and [x] for complete;
make the next phase unambiguous. Record compatibility, rollback, and security
assessment needs in the risks.

Context must explain existing behavior, the problem, and constraints that
affect implementation choices. Exclude planning history, inspection logs,
generic disclaimers, and repeated goals. Keep it brief when little background
is needed.

Check the plan's accuracy, links, structure, and dependencies. Record tests and
builds as future implementation validation; do not run them merely to generate
the plan or satisfy its review checkpoint. Run an executable check only for an
explicit user request, a repository rule covering the changed files, or a
concrete uncertainty that affects the plan. Explain that reason, choose the
narrowest relevant check, and distinguish its results from proposed validation.

Keep the resulting plan sized for one focused, self-contained PR. Each phase
must deliver one coherent outcome and include only the smallest necessary
implementation, tests, configuration, and documentation. Split a phase when
it combines multiple user outcomes, independently deployable or revertible
changes, separate ownership, an optional enhancement, or a prerequisite that
can merge independently. Do not use phases to group unrelated cleanup or
future work, and do not let a phase depend on undocumented work hidden in
another phase. During planning, perform a PR-size check: if the scope would
produce a broad or difficult-to-review diff, narrow the phase or create
explicitly dependent plans/phases with clear handoffs and acceptance criteria.
Keep each phase's commits focused on that phase's outcome.

After user confirmation, write `README.md` and `phase-N.md` under a
date-slugged directory beneath the installed Markdown output policy's
`ticket-to-plan` skill directory. For chat-only input, retain the retrospective
ticket in that same skill directory. An explicit output path wins. Run
[plan-review](../plan-review/SKILL.md) as the checkpoint. Stop at
ready-to-implement; never write application code.

Start each generated `README.md` with a concise Mermaid diagram that gives the
overview of the work and shows phase dependencies before the detailed Goal,
Context, Phases, and Risks sections. Keep node and edge descriptions brief but
understandable, and keep the diagram synchronized with the phase order and
dependency statements.

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
