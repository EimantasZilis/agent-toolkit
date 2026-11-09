# ticket-to-plan: turn a ticket into an implementation plan

[All skill examples](../../EXAMPLES.md)

[ticket-to-plan](../../.agents/skills/scoping/ticket-to-plan/SKILL.md) turns a ticket
into scoped implementation units and validation criteria based on the
repository. It confirms scope before writing and reviewing the plan, then
stops before implementation.

## Why use it

Start implementation with clear scope, dependencies, and validation. The plan
breaks the ticket into reviewable phases and commits, giving later sessions a
shared record of what to build and check.

## Example prompt

> Use ticket-to-plan for this ticket: Customers need to download their invoices
> as PDFs. Only the invoice owner may download one. Keep the existing invoice
> page behavior intact and exclude bulk downloads.

## Illustrative result

Each ticket gets its own planning folder with a `README.md` and one file per
phase. For the invoice ticket, the folder could contain:

```text
<repo-root>/<branch-slug>/ticket-to-plan/2026-11-09-invoice-pdf-download/
├── README.md
└── phase-1.md
```

The `README.md` is the ticket's planning index. It starts with a Mermaid diagram
showing the work and phase dependencies, followed by:

- Goal: what the ticket must achieve.
- Context: existing behavior, relevant code paths, and implementation constraints.
- Out of scope: work excluded from this ticket.
- Assumptions: decisions that still need confirmation or supporting evidence.
- Phases: links to phase files, their status, and the next phase to implement.
- Risks: compatibility, rollback, and any security assessment needs.

Each phase delivers a reviewable outcome and can become a separate PR for the
ticket. It is broken into self-contained commits with clear scope, acceptance
criteria, and validation checks. Dependencies show what to implement and merge
first, so you can work through the ticket in manageable steps.

For the invoice ticket, `phase-1.md` could include:

```markdown
# Phase 1: Owner-only invoice PDF download

Outcome: Invoice owners can download a PDF from the existing invoice page.

## Commits

### [ ] Commit 1: Add the authorized PDF download endpoint

- Reuse the existing invoice ownership check before generating a PDF.
- Return a PDF containing the authorized invoice's details.
- Validate owner access, denied access, and PDF contents with endpoint tests.

### [ ] Commit 2: Expose the download on the invoice page

- Add a link to the endpoint from Commit 1.
- Preserve existing invoice page behavior.
- Validate the link with page tests and a manual download.

## Phase exit checks

- Owners can download their invoices through the page.
- Other customers cannot download those invoices.
- Existing invoice page behavior remains intact.
```

The generated plan names the repository's actual code paths and validation
commands. After scope confirmation, the skill writes the files and runs the
plan-review as a sanity check.
