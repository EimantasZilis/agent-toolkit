# commit-message: summarize staged changes

[All skill examples](../../EXAMPLES.md)

[commit-message](../../.agents/skills/productivity/commit-message/SKILL.md) turns a
staged diff into a short, copy-ready message. It keeps unrelated unstaged edits
and planned work out of the summary.

## Why use it

Spend less time drafting commit messages while keeping Git history accurate.
The summary follows the staged diff, so unrelated edits and future plans stay
out of the record.

## Example prompt

```text
<commit-message>
```

## Illustrative result

For a staged change adding owner-only invoice PDF downloads:

```text
Add owner-only invoice PDF downloads
```

The skill does not stage files or create a commit. If nothing is staged, it
reports that and stops.
