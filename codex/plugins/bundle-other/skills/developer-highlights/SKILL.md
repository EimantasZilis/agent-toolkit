---
name: developer-highlights
description: >-
  Build a concise, evidence-based career narrative from an engineer's git
  history across confirmed workspace repositories. Use for portfolio, review,
  or onboarding retrospectives; invoke explicitly and confirm scope first.
---

# Contribution story

Turn confirmed contribution history into a narrative rather than a commit dump.

## Required result

Give a compact activity snapshot, grouped technology stack, five to ten thematic
achievements, and three to six user or business outcomes. Use approximate counts
when helpful, omit per-repository tables, and never invent metrics or technologies.

## Workflow

1. Confirm the person, matching names or email addresses, and all repositories to scan.
2. Skip missing or non-git paths and note them in a short scope note.
3. Gather dates, commit volume, authored file types, manifests, durable themes,
   migrations, security work, and product outcomes. Ignore merge, dependency,
   and lint noise unless it marks a meaningful transition.
4. Merge evidence across repositories, rank sustained ownership above one-off work,
   and distinguish observed facts from inferred impact.
5. Present the result as At a glance, Tech stack, Career highlights, and
   Business / user impact.

Lead with the strongest evidence-based themes. Use short paragraphs or bullets,
state each fact once, and omit repeated context and commit-by-commit narration.
Keep technical details only when they substantiate an achievement or outcome.
Preserve uncertainty where it changes the reader's interpretation.

Do not write a file unless the user requests one. When a file is requested and
no path is supplied, use the installed Markdown output policy beneath the
`developer-highlights` skill directory. An explicit output path wins.

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
