# Ruby logging and observability

## Apply this first

Make important operations observable without exposing secrets or personal data.
Prefer stable event names and small structured fields over object dumps.

- Log at the boundary that owns the decision and avoid duplicate messages.
- Include operation, outcome, correlation context, and safe identifiers when
  they materially help diagnosis.
- Use the repository's configured logger and instrumentation API, not `puts`.
- Redact credentials, tokens, authorization material, and sensitive attributes.
- Include exception class and safe context for failures; preserve the cause in
  the error path rather than serializing an entire exception blindly.

```ruby
# DO: emit stable, safe context.
logger.info("article.published", article_id: article.id)
```

```ruby
# DO NOT: dump arbitrary input or a full request into logs.
logger.info(params)
```

Allowed exceptions: temporary local debugging may use extra output while
developing, but it must not ship. A regulated audit event may retain more
context only with explicit access, retention, and redaction controls.

## Quick checklist

- [ ] Events identify operation and outcome without noisy duplication.
- [ ] Correlation context is present where supported.
- [ ] Secrets and sensitive attributes are filtered.
- [ ] Failure logs preserve useful context safely.

## Sources and decisions

- [Ruby Logger documentation](https://ruby-doc.org/stdlib-3.0.0/libdoc/logger/rdoc/Logger.html), accessed 2026-09-30.
- Logging field names and retention remain repository and operations concerns.

