# Go logging and observability

## Apply this first

- Use the repository's logging and telemetry APIs consistently. For new
  standard-library code on Go 1.21+, prefer `log/slog` when it fits the local
  architecture.
- Emit structured attributes with stable keys, useful severity, and enough
  operation context to correlate a failure. Avoid concatenating dynamic data
  into messages when a field is more useful.
- Never log passwords, tokens, session identifiers, private keys, or raw
  sensitive payloads. Redact or hash identifiers according to local policy.
- Do not log and return the same error at every layer. Choose the layer that
  owns the user-visible or operational decision and attach context elsewhere.
- Keep logs bounded and actionable; use metrics and traces for counts,
  durations, and distributed request relationships.

~~~go
logger.ErrorContext(ctx, "charge failed",
    slog.String("account_id", redact(accountID)),
    slog.Any("error", err),
)
~~~

## Quick checklist

- [ ] Log fields use stable names and appropriate levels.
- [ ] Sensitive data is excluded or redacted before logging.
- [ ] Context, operation, and correlation information are available.
- [ ] Errors are not duplicated at every stack layer.
- [ ] Tests cover redaction and important observability behavior.

## DO / DO NOT

~~~go
// DO: make the structured field explicit and safe.
logger.InfoContext(ctx, "user signed in", slog.String("user_id", hash(id)))

// DO NOT: put credentials or arbitrary request bodies in logs.
logger.Debug("request", "authorization", token, "body", body)
~~~

## Allowed exceptions

Local development may enable verbose logging with synthetic data. Incident
debugging may temporarily increase detail only through approved, access-
controlled configuration; scrub captured output before retaining or sharing it.

## Sources

- [log/slog package](https://pkg.go.dev/log/slog) (accessed 2026-09-30)
- [Structured Logging with slog](https://go.dev/blog/slog) (accessed 2026-09-30)
- [Go security best practices](https://go.dev/doc/security/best-practices) (accessed 2026-09-30)
