---
name: annotate
description: Explain only a branch's changed code inline in terminal output, after establishing whether the scope is uncommitted work or committed branch changes.
---

# Annotate branch changes

Annotate code changes without changing any files. The output is the artifact:
never add comments to the source, stage files, commit, or otherwise mutate the
working tree.

## Establish the scope first

Run a read-only status check before inspecting the diff:

```text
git status --short --branch
```

If there are any staged or unstaged uncommitted changes, stop and ask the user
whether to apply the annotation only to the unstaged changes. Do not inspect or
annotate code until the user answers. Use the answer as follows:

- If the user says yes, scope is the unstaged working-tree diff (`git diff`).
  Exclude staged changes, even when the same file also has unstaged edits.
- If the user says no, scope is all uncommitted changes (`git diff HEAD`),
  including staged and unstaged changes.

If there are no uncommitted changes, scope is the committed changes on the
current branch. Identify the branch base without changing branches: prefer the
configured upstream's merge-base with `HEAD`; if no upstream exists, use the
merge-base with the repository's default branch (`origin/HEAD`, `main`, or
`master`, in that order). Annotate the range from that base to `HEAD`.

State the selected scope and exact diff range before displaying annotations.
If the requested base cannot be identified, report the blocker rather than
guessing or widening the scope.

## Inspect and display the affected code

Use the selected diff to identify changed files and hunks. Read only the
affected code needed to explain those hunks; do not summarize unchanged files
or unrelated surrounding code. Include additions, modifications, and relevant
deletions. For a deleted file or deleted block, display the pre-change version
from the diff's old side and explain the removed behavior.

For every affected file, print a filename heading. Display each affected hunk
with stable source line numbers and a clear code fence or equivalent terminal
layout, for example:

```text
FILE: src/example.py
LINES: 18-27

18 | changed code
19 | changed code
20 | nearby context

COMMENT: High-level explanation of the small changed block.

```

Use the post-change line numbers for added or modified code. For deleted code,
use the old-file line numbers and label them as such. Keep context minimal but
large enough to make each explanation understandable. Preserve the source
text exactly apart from the display gutter; annotations must be visibly
separate from the code and must not look like source comments.

When the output is being emitted to a terminal that preserves ANSI styling,
make the display easier to scan with colour:

- Prefer an already-installed syntax highlighter for the affected language,
  with colour forced on and without changing the source text. Do not install a
  dependency just for annotation.
- If no syntax highlighter is available, use ANSI SGR styling for display
  structure only: dim line-number gutters, distinct filename and range
  headings, and a contrasting `COMMENT:` label. Use addition/deletion colours
  only when the displayed layout explicitly distinguishes those states.
- Reset styling after every line or display block. Never leave raw escape codes
  in a Markdown or captured-log fallback, and never rely on colour alone to
  convey meaning.
- If ANSI styling is unsupported or would be shown literally, use a
  language-tagged code fence or the plain terminal layout above.

## Write the annotations

Add a concise `COMMENT:` after each changed line or small coherent block. Each
comment should explain what the change does at a high level and, when useful,
its role in the surrounding flow. Prefer one comment for a coherent block over
repeating the same explanation on every line. Do not review quality, propose
fixes, or explain unchanged implementation details unless needed to understand
the changed code.

Before finishing, verify that every displayed annotation belongs to the
selected diff and that no file was modified:

```text
git status --short
```
