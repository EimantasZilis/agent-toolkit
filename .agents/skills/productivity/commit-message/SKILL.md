---
name: commit-message
description: >-
  Write a short, copy-ready commit message from staged Git changes and recent
  conversation context when available. Use when asked to draft a commit message.
---

# Write a commit message

Inspect the target repository's staged changes with `git diff --cached --stat`
and `git diff --cached`. Read enough of the staged diff to understand all changes,
including additions, deletions, and renames. For partially staged files, describe
only the staged version; do not include unstaged or untracked work.

Use recent conversation context, when available, to explain the purpose of the
changes. Treat the staged diff as the source of truth: omit planned work,
abandoned approaches, and claims the staged changes do not support. Do not
require conversation history or search private session logs when context is
unavailable.

Follow explicit user instructions and repository commit-message conventions.

Return one suggested message in a plain text code block with no surrounding
commentary. Default to a single subject line; add a brief body only when needed
to explain a material reason or consequence that the subject cannot convey.
Do not invent motivation or validation results. If nothing is staged, state
that there are no staged changes and stop.

Drafting a message does not authorize staging files or creating a commit.
