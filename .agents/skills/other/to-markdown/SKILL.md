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
- Write the file in the current workspace unless the user specifies another
  location. Avoid overwriting an existing file without the user's instruction;
  choose an unused descriptive variant such as `topic-2.md`.

## Result

Write the file, then report its exact path. Do not paste the entire response
again unless the user asks for it.
