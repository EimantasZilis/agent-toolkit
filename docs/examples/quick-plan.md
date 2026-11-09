# quick-plan: decide whether an idea is worth building

[All skill examples](../../EXAMPLES.md)

[quick-plan](../../.agents/skills/scoping/quick-plan/SKILL.md) helps you decide
whether to invest in an idea before spending time on tickets or implementation.
It checks the evidence needed to make that decision and gives one recommendation
with reasons, assumptions, and a clear next step.

## Why use it

Resolve the unknowns that could make an idea unworkable before investing in
tickets or implementation. You get one recommendation backed by evidence,
with assumptions and a clear next step.

## Example prompt

> Use quick-plan to investigate whether our existing PDF renderer can support
> customer invoice downloads. Check deployment constraints and the current
> rendering path. Exclude bulk downloads and do not implement anything.

## Illustrative result

The result connects the evidence to the decision. For the invoice feature, it
could look like this:

> **Findings**
>
> - The current rendering path supports the invoice's required layout.
> - The deployment image already includes the renderer's runtime dependencies.
> - Renderer failures still need defined behavior in the download flow.
>
> **Assumption:** Downloads cover one invoice at a time; bulk generation is
> outside this investigation.
>
> **Recommendation: requirements-to-tickets.** The existing renderer is suitable
> for this scope. Create an owner-only download ticket and include renderer
> failure handling in its acceptance criteria.

You can now scope work around the existing renderer with the deployment question
resolved and a concrete failure case to cover. If the evidence instead showed a
blocking constraint, the recommendation could be to abandon the approach before
investing in a build plan.
