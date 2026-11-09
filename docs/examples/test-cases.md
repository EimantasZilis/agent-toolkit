# test-cases: prepare manual test cases

[All skill examples](../../EXAMPLES.md)

[test-cases](../../.agents/skills/scoping/test-cases/SKILL.md) produces executable
manual checks for changed flows.

## Why use it

Make manual testing repeatable and easier to hand off. Prioritized cases define
the setup, actions, and expected results, helping testers focus on important
risks with less guesswork.

## Example prompt

> Use test-cases for this branch against main and the PDF ticket at
> `<ticket-file>`. Our local test environment has two customer accounts with
> one invoice each. Focus on the download flow and ownership boundary.

## Illustrative result

Prioritized cases with setup, actions, expected results, and gaps.
Case excerpt:

- **Priority:** P0 — access control.
- **Setup:** In the local test environment, customers A and B each own an invoice.
  Record B's invoice download URL using the test fixture or B's session.
- **Actions:** Sign in as A and request B's invoice download URL.
- **Expected:** Access is denied and no PDF or invoice details are returned.
- **Evidence:** Record the response status and confirm that the body contains no
  invoice data. End the test session afterward.

Run the cases against the intended build before recording pass/fail results.
