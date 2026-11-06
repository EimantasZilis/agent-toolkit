# Go documentation

## Apply this first

- Document exported packages, types, functions, methods, constants, and
  variables with comments that begin with the declared name and describe the
  contract, not the implementation.
- Keep package comments adjacent to the package clause. Explain purpose,
  important invariants, concurrency safety, ownership, and error behavior
  where callers need that information.
- Make examples compile and show the supported path. Update documentation in
  the same change as behavior or public API changes.
- Prefer clear names and examples over comments that restate obvious code.
  Comments must remain accurate after refactoring.

~~~go
// Client sends requests to the billing service.
// Client is safe for concurrent use.
type Client struct { /* ... */ }

// Charge records an amount for accountID and returns the service response.
func (c *Client) Charge(ctx context.Context, accountID string, amount Money) error {
    // ...
    return nil
}
~~~

## Quick checklist

- [ ] Exported declarations have accurate, name-leading documentation.
- [ ] Public errors, nil behavior, blocking, and concurrency guarantees are stated.
- [ ] Examples compile or are covered by the repository's example tests.
- [ ] Comments explain decisions and contracts rather than syntax.
- [ ] Docs, changelog, and migration notes match the public behavior change.

## DO / DO NOT

~~~go
// DO: document the observable contract.
// Parse accepts a UTF-8 value and returns ErrInvalid for malformed input.

// DO NOT: repeat the implementation without telling callers what to expect.
// Parse parses the input.
~~~

## Allowed exceptions

Unexported helpers need no comment when their name and local use are clear. A
generated file may carry generator-owned comments and be excluded from manual
documentation checks, but its source template or generator must be maintained.

## Sources

- [Go doc comments](https://go.dev/doc/comment) (accessed 2026-09-30)
- [Go Code Review Comments: comment sentences](https://go.dev/wiki/CodeReviewComments#comment-sentences) (accessed 2026-09-30)
- [Go examples](https://go.dev/blog/examples) (accessed 2026-09-30)
