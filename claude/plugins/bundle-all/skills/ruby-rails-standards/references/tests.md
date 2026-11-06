# Rails tests

## Apply this first

Test behavior at the lowest useful boundary and keep tests deterministic. Use
the repository's existing framework (Minitest, RSpec, or another configured
runner) rather than introducing a second style.

- Cover business rules with focused unit/model tests and HTTP contracts with
  request/integration tests.
- Use fixtures/factories/builders consistently with the local suite; avoid
  hidden global state and order dependence.
- Assert observable outcomes and meaningful side effects, not private method
  call counts unless the interaction is the contract.
- Test authorization, invalid input, transaction failure, retries, and
  idempotency when changed behavior crosses those boundaries.
- Reserve browser/system tests for critical user paths and complex JavaScript;
  prefer faster integration tests for ordinary workflows.

```ruby
# DO: assert the user-visible contract and persistence outcome.
test "publishing an article makes it visible" do
  article = articles(:draft)

  patch publish_article_path(article), params: { article: { title: "New" } }

  assert_redirected_to article_path(article)
  assert_predicate article.reload, :published?
end
```

```ruby
# DO NOT: make a broad system test for every validation branch or assert
# implementation details that can change without changing behavior.
```

Allowed exceptions: a system test is appropriate for a critical end-to-end
flow, browser-only behavior, or a regression that lower-level tests cannot
reproduce. An interaction assertion is appropriate when avoiding the call is a
correctness guarantee, such as not enqueueing before commit.

## Quick checklist

- [ ] The test follows the local runner, naming, fixture, and helper patterns.
- [ ] Setup, action, and assertions are easy to distinguish.
- [ ] Success, failure, authorization, and boundary cases are covered.
- [ ] Tests do not depend on time, randomness, network, or execution order.
- [ ] System-test coverage is limited to user-critical or browser-specific work.

## Sources and decisions

- [Testing Rails Applications](https://guides.rubyonrails.org/testing.html), accessed 2026-09-30.
- [Active Job Basics](https://guides.rubyonrails.org/active_job_basics.html), accessed 2026-09-30.
- Rails explicitly describes system tests as slower and more brittle than lower
  level tests; the routing rule preserves that trade-off while allowing local
  test-framework conventions.

