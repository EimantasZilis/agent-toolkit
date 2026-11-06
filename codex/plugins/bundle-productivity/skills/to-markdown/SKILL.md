---
name: to-markdown
description: >-
  Save the immediately preceding response as a Markdown file while preserving
  its structure and formatting. Use when the user asks to export, save, or
  turn the previous response into a Markdown file.
---

# To Markdown

Create a `.md` file from the immediately preceding assistant response.

## Content

- Use the complete preceding response as the source, including headings,
  paragraphs, lists, tables, links, code fences, blockquotes, emphasis, and
  intentional blank lines.
- Preserve the response's Markdown structure and meaning. Do not summarize,
  rewrite, reorder, or add an introductory wrapper.
- If the response contains formatting represented by rich text rather than
  Markdown, translate it to the closest standard Markdown equivalent.
- Keep code blocks fenced and preserve their language labels when present.
- Preserve links as links and use Markdown image syntax for inline images when
  the source provides an address or an accessible local path. If an image
  cannot be embedded, retain its alternative text and source reference.

## Filename and location

- If the user gives a filename or path, use it and add the `.md` extension when
  the requested filename has no extension. Respect an explicitly supplied
  non-`.md` extension only when the user clearly requests it.
- Otherwise choose a short, descriptive, filesystem-safe lowercase filename
  based on the response's main subject, such as `database-migration-plan.md`.
  Use `response.md` when no subject can be inferred.
- Write the file beneath the installed Markdown output policy's `to-markdown`
  skill directory unless the user specifies another location. Avoid
  overwriting an existing file without the user's instruction; choose an
  unused descriptive variant such as `topic-2.md`.

## Result

Write the file, then report its exact path. Do not paste the entire response
again unless the user asks for it.

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
