# Ruby and Rails documentation

## Apply this first

Document the contract that a maintainer or caller must know: purpose,
preconditions, side effects, failure behavior, security assumptions, and
configuration. Keep comments close to the code they explain and prefer names
and tests over comments that restate syntax.

- Document public Ruby APIs and non-obvious service/job contracts with concise
  YARD-compatible comments when the repository uses YARD or another format.
- Document routes, request parameters, response/error shapes, authorization,
  retries, and idempotency for public HTTP or job boundaries.
- Update README, changelog, API schema, or runbook material when behavior or
  operations change; do not leave stale examples.
- Explain why for compatibility workarounds, unusual queries, callbacks,
  security exceptions, and performance trade-offs.

```ruby
# DO: document the boundary contract, not every line.
# Publishes the article once. Safe to retry after a timeout because the
# publication transition is guarded by the database state.
class PublishArticle
  def call(article_id, actor_id:)
    # ...
  end
end
```

```ruby
# DO NOT: leave a comment that merely repeats the code.
article.update!(published_at: Time.current) # update published_at
```

Allowed exceptions: private, self-explanatory methods need no docstring;
generated API docs or schema files may be the canonical contract. Keep a
comment when removing it would make a security, compatibility, or concurrency
decision easy to undo accidentally.

## Quick checklist

- [ ] Changed public behavior and operational assumptions are documented.
- [ ] Examples use current names, parameters, routes, and commands.
- [ ] Comments explain rationale and constraints, not mechanics.
- [ ] Security, retry, transaction, and idempotency contracts are explicit.
- [ ] Documentation checks or link validation pass when configured.

## Sources and decisions

- [Rails Guides](https://guides.rubyonrails.org/), accessed 2026-09-30.
- [Ruby documentation](https://ruby-doc.org/), accessed 2026-09-30.
- Official framework documentation is the source for public Rails behavior;
  repository API schemas and runbooks override generic examples.

