# Rails logging and observability

## Apply this first

Make important state transitions observable without exposing secrets or
personal data. Prefer structured, stable fields and a useful event name over
dumping whole objects.

- Log at the boundary that owns the decision; avoid duplicate logs at every
  layer.
- Include correlation/request/job identifiers, operation, outcome, and safe
  resource identifiers when they materially help diagnosis.
- Use the configured Rails logger and notification/instrumentation APIs rather
  than `puts` or ad-hoc output.
- Redact credentials, session material, authorization headers, payment data,
  and sensitive attributes; configure parameter filtering as defense in depth.
- Keep normal control flow at an appropriate level and include exception class,
  message, and cause only when safe.

```ruby
# DO: emit a small, searchable event with safe context.
Rails.logger.info(
  event: "article.published",
  article_id: article.id,
  actor_id: current_user.id
)
```

```ruby
# DO NOT: serialize a request, model, or exception containing secrets.
Rails.logger.info(params.to_unsafe_h)
Rails.logger.error(error.full_message)
```

Allowed exceptions: temporary local debugging may use extra output while
developing, but it must not ship; a regulated audit event may retain more
context only when its access, retention, redaction, and purpose are explicitly
controlled.

## Quick checklist

- [ ] Events identify the operation and outcome without noisy duplication.
- [ ] Correlation context is present where supported.
- [ ] Secrets and sensitive attributes are filtered before logging.
- [ ] Failure logs preserve actionable exception context safely.
- [ ] Metrics/notifications are emitted at stable business boundaries.

## Sources and decisions

- [Rails Security Guide](https://guides.rubyonrails.org/security.html), accessed 2026-09-30.
- [Active Support Instrumentation](https://guides.rubyonrails.org/active_support_instrumentation.html), accessed 2026-09-30.
- Rails supplies logging, filtering, and instrumentation primitives; exact event
  names and retention are repository and operations concerns.

