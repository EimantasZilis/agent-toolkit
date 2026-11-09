# pr-description: write a useful PR description

[All skill examples](../../EXAMPLES.md)

[pr-description](../../.agents/skills/productivity/pr-description/SKILL.md) drafts a
copy-ready Markdown description explaining why the change matters and how it
was validated. 

## Why use it

Help reviewers understand the change without reconstructing its purpose from
the diff. The description connects the user problem, delivered behavior, and
validation, reducing the work of preparing each PR.

## Example prompt
For a given set of changes that allow:
 - Invoice owners can now download a PDF from the invoice page. Previously they had to request one from support.
 - The endpoint checks ownership before generating the PDF

Call the skill via:

```text
<pr-description>
```

You can also provide additional context or any areas that you wish to highlight in the PR description.

## Illustrative result

A description explaining the problem, cause, and fix:

```markdown
This MR lets customers download their own invoices as PDFs.

## The problem

Customers had to contact support to get a PDF invoice.

## The cause

The invoice page offered no PDF download.

## The fix

Add a download link and check ownership before generating the PDF. Owner and
non-owner access tests passed.
```
