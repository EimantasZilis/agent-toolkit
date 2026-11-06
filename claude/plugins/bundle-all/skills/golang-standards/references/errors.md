# Go errors and resource cleanup

## Apply this first

- Return errors as values for expected failures; make the caller decide whether
  to retry, translate, report, or ignore them.
- Add context while preserving identity with `%w`; inspect wrapped errors with
  `errors.Is` and `errors.As`, not string matching.
- Handle each error deliberately. Do not discard a meaningful error with `_`
  unless the operation is explicitly best-effort and the reason is clear.
- Close resources at the point ownership begins, use `defer` after successful
  acquisition, and preserve a more important primary error if cleanup also
  fails.
- Use panic only for unrecoverable programmer or initialization failures at a
  clearly documented boundary; do not use it for routine input or I/O errors.

~~~go
if err := decoder.Decode(&out); err != nil {
    return fmt.Errorf("decode response: %w", err)
}
if errors.Is(err, context.DeadlineExceeded) { /* retry or translate */ }
~~~

## Quick checklist

- [ ] Every expected failure is returned or intentionally handled.
- [ ] Wrapped errors preserve a stable sentinel or typed error contract.
- [ ] Cleanup is registered immediately after successful acquisition.
- [ ] Retries are bounded, cancellable, and safe for the operation.
- [ ] Panic and fatal exits are limited to documented process boundaries.

## DO / DO NOT

~~~go
// DO: preserve the cause and add operation context.
return fmt.Errorf("load account %q: %w", id, err)

// DO NOT: make callers parse unstable text.
return errors.New("load account failed: " + err.Error())
~~~

## Allowed exceptions

Use a package-specific error type or sentinel when callers need stable
classification. A cleanup error may replace the return error only when cleanup
is the operation's primary contract; otherwise record it through the local
logging or multi-error policy.

## Sources

- [Error handling and Go](https://go.dev/blog/go1.13-errors) (accessed 2026-09-30)
- [errors package](https://pkg.go.dev/errors) (accessed 2026-09-30)
- [Effective Go: panic and recover](https://go.dev/doc/effective_go#panic) (accessed 2026-09-30)
