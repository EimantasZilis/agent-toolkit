# Go core

## Apply this first

- Prefer simple, idiomatic packages with narrow responsibilities and names
  that read naturally at the call site; avoid `util`, `common`, and catch-all
  packages.
- Format every Go file with `gofmt`. Use short declarations where they improve
  clarity, keep control flow flat, and return early for error cases.
- Design interfaces at the consumer boundary and keep them small. Accept
  interfaces when callers need substitution; return concrete types when the
  implementation is the useful result.
- Make ownership, nil behavior, mutation, and concurrency expectations clear.
  Prefer zero values that are useful and constructors only when invariants need
  enforcement.
- Pass `context.Context` explicitly as the first parameter for request-scoped
  work; do not store it in a struct or use it as an optional data bag.

~~~go
func (s *Store) Find(ctx context.Context, id string) (Record, error) {
    if err := ctx.Err(); err != nil {
        return Record{}, err
    }
    return s.records.Load(ctx, id)
}
~~~

## Quick checklist

- [ ] Package and exported names are concise and idiomatic.
- [ ] The zero value and ownership rules are safe or documented.
- [ ] Interfaces are small and defined by their consumers.
- [ ] Error paths return early and resource ownership is visible.
- [ ] `gofmt` has been run and behavior is covered by relevant tests.

## DO / DO NOT

~~~go
// DO: define the smallest interface where it is consumed.
type Clock interface { Now() time.Time }

// DO NOT: create a broad interface solely because a type has many methods.
type Everything interface { /* dozens of unrelated methods */ }
~~~

## Allowed exceptions

Use a larger or producer-owned interface when an external API requires that
shape, but isolate it behind an adapter and explain the dependency. Store a
context only when a third-party interface forces the signature; do not make
that exception a general application pattern.

## Sources

- [Effective Go](https://go.dev/doc/effective_go) (accessed 2026-09-30)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) (accessed 2026-09-30)
- [How to Write Go Code](https://go.dev/doc/code) (accessed 2026-09-30)
