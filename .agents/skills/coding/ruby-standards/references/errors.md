# Ruby errors

## Apply this first

Handle errors at the boundary that can make a correct decision. Preserve the
original exception and context, distinguish expected failure from programming
or infrastructure failure, and make retries safe.

- Rescue the narrowest specific exception you can handle; never use a bare
  rescue in application or library code.
- Do not rescue, log, and continue when state is incomplete or unknown.
- Use explicit result handling or exceptions consistently with the caller's
  contract; do not silently ignore `false` or `nil` failure values.
- Retry only identified transient failures with bounded backoff and an
  idempotent operation.
- Use `ensure` for cleanup that must happen on every exit path.

```ruby
# DO: translate an expected boundary failure and preserve its cause.
def fetch!
  client.fetch!
rescue Client::TimeoutError => error
  raise Unavailable, "client timed out", cause: error
end
```

```ruby
# DO NOT: hide all failures or retry a non-idempotent operation forever.
rescue StandardError
  retry
end
```

Allowed exceptions: a top-level process boundary may catch broad exceptions to
report and terminate according to its contract, but it must preserve context,
avoid sensitive data, and not pretend the operation succeeded.

## Quick checklist

- [ ] Rescues name failures the code can actually handle.
- [ ] Unexpected failures remain observable and preserve their cause.
- [ ] Retry policy is bounded, selective, and idempotent.
- [ ] Cleanup is correct on every exit path.
- [ ] Tests cover expected failure and recovery behavior.

## Sources and decisions

- [Ruby exception processing and `ensure`](https://ruby-doc.org/docs/ruby-doc-bundle/UsersGuide/rg/ensure.html), accessed 2026-09-30.
- Ruby documents `ensure` for unconditional cleanup; retry policy remains
  application-specific and must be explicit.

