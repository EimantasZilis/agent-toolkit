# Ruby and Rails errors

## Apply this first

Handle errors at the boundary that can make a correct decision. Preserve the
original exception and context, distinguish expected domain failure from a
programming or infrastructure failure, and make retries safe.

- Rescue the narrowest specific exception you can handle; never use a bare
  rescue in application code.
- Do not rescue, log, and continue when the operation is incomplete or state is
  unknown. Re-raise or return an explicit failure.
- Use bang methods or explicit result handling consistently with the caller's
  contract; do not silently ignore `false` or validation errors.
- Retry only transient, identified failures with bounded backoff and an
  idempotent operation. Do not retry validation, authorization, or programming
  errors.
- Ensure cleanup with `ensure`, and use transaction rollback semantics rather
  than manually guessing which writes succeeded.

```ruby
# DO: handle the expected failure and preserve unexpected failures.
def charge!
  gateway.charge!(amount)
rescue PaymentGateway::TimeoutError => error
  raise ChargeUnavailable, "payment gateway timed out", cause: error
end
```

```ruby
# DO NOT: hide all failures or retry a non-idempotent write indefinitely.
rescue StandardError
  retry
end
```

Allowed exceptions: a top-level job or request boundary may catch a broad
exception to report and terminate/retry according to the framework contract;
it must preserve context, avoid leaking sensitive data, and not pretend the
operation succeeded. A compatibility boundary may translate exception classes
when the mapping is documented and tested.

## Quick checklist

- [ ] Rescues name the failure the code can actually handle.
- [ ] Unexpected failures remain observable and preserve their cause.
- [ ] Retry policy is bounded, selective, and idempotent.
- [ ] Cleanup and transaction behavior are correct on every exit path.
- [ ] Tests cover expected failure and recovery behavior.

## Sources and decisions

- [Ruby exception processing and `ensure`](https://ruby-doc.org/docs/ruby-doc-bundle/UsersGuide/rg/ensure.html), accessed 2026-09-30.
- [Active Job Basics](https://guides.rubyonrails.org/active_job_basics.html), accessed 2026-09-30.
- Ruby documents `ensure` for unconditional cleanup; Rails documents job
  callbacks and enqueue timing. Retry limits and exception taxonomy remain
  application-specific and must be explicit.

