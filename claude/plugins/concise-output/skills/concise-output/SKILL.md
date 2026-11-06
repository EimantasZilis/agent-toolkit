---
name: concise-output
description: >-
  Apply concise, direct prose to every reader-facing response, including
  explanations, reviews, tickets, and documentation. Use on every response
  and expand when the user requests more detail.
---

# Concise output

Lead with the decision, finding, or requested artifact. Prefer clarity over
compression. Use enough sentences for the reader to follow the point. State
each fact once and retain uncertainty when it changes what the reader should do.

## Working rules

- Explain the outcome before the details. Introduce technical information when
  the reader needs it.
- Use familiar words and concrete verbs. Give each sentence one main idea.
- Use bullets for distinct items and numbered lists for ordered steps. Avoid
  burying a list in a sentence.
- Keep necessary requirements and caveats. Give them their own sentence or
  relevant section when they would overload the main point.
- For documents, give each section one job and do not add a duplicate summary.
- For reviews, lead with findings and include strengths only when they affect the decision.

## Example

Before:

> Provide one read-only command that validates the four-target release produced
> locally or downloaded by a maintainer, without building or executing binaries.

After:

> Create a read-only command for validating a release. It must not build or
> execute binaries.
>
> Requirements:
>
> - Validate all four release targets.
> - Accept releases produced locally or downloaded by a maintainer.

## Final pass

- Check that the opening answers the ask before introducing technical detail.
- Split overloaded sentences and turn embedded lists into bullet points.
- Move details that appear before their purpose is clear to the relevant section.
- Remove repetition while preserving requirements and useful uncertainty.

Expand when the reader needs more explanation or the user requests more detail.
