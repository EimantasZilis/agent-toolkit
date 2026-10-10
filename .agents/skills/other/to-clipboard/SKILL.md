---
name: to-clipboard
description: >-
  Copy the immediately preceding response to the system clipboard as Markdown
  while preserving its structure and formatting. Use when the user asks to
  copy or put the previous response on the clipboard.
---

# To Clipboard

Copy the complete immediately preceding assistant response to the system
clipboard as Markdown.

## Content

- Preserve headings, paragraphs, lists, tables, links, code fences, blockquotes,
  emphasis, images, and intentional blank lines.
- If the response is represented as rich text rather than Markdown, translate
  it to the closest standard Markdown equivalent before copying.
- Keep code fences and their language labels intact.
- Preserve links as Markdown links. Preserve image alternative text and source
  references using Markdown image syntax where possible.
- Do not summarize, rewrite, reorder, or add an introductory wrapper.

## Clipboard operation

- Use the available native or environment-provided clipboard mechanism.
- Copy the Markdown source itself, not rendered text or HTML-only formatting,
  so it can be pasted directly into a Markdown file later.
- Do not write a file or alter unrelated clipboard content if the copy cannot
  be completed.
- If clipboard access is unavailable or fails, report the failure clearly and
  provide the Markdown content in the response so the user has a recoverable
  fallback.

## Result

After a successful copy, briefly confirm that the Markdown content is on the
clipboard. Do not paste the entire response again unless the user asks for it.
