# Ruby tests

## Apply this first

Test behavior at the lowest useful boundary and keep tests deterministic. Use
the repository's existing framework and naming conventions rather than adding a
second test style.

- Keep setup, action, and assertions easy to distinguish.
- Assert observable outcomes and meaningful side effects, not implementation
  calls unless the interaction is the contract.
- Isolate network, time, randomness, filesystem, and process boundaries with
  the repository's established helpers.
- Cover invalid input, expected failures, boundary values, and idempotency
  where they affect behavior.
- Keep fixtures and test doubles minimal, local, and readable.

```ruby
# DO: assert the public result and the important side effect.
it "publishes an article" do
  result = described_class.new(article).call

  expect(result).to be_success
  expect(article.reload).to be_published
end
```

```ruby
# DO NOT: assert a private helper call when the result is the contract.
expect(article).to receive(:set_published_flag)
```

Allowed exceptions: interaction assertions are appropriate for an external
boundary, security guarantee, or performance contract; integration tests are
appropriate when collaboration between objects is the behavior under test.

## Quick checklist

- [ ] The test follows the local runner, helpers, and naming patterns.
- [ ] Success, failure, and meaningful boundary cases are covered.
- [ ] Tests are isolated from time, randomness, network, and order.
- [ ] Assertions focus on behavior and useful diagnostics.

## Sources and decisions

- [Ruby documentation](https://ruby-doc.org/), accessed 2026-09-30.
- [RuboCop RSpec extension](https://docs.rubocop.org/rubocop-rspec/), accessed 2026-09-30.
- The pack leaves the test framework choice to the repository because Ruby
  projects commonly use Minitest or RSpec.

