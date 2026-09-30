# Ruby documentation

## Apply this first

Document contracts that callers and maintainers must know: purpose,
preconditions, side effects, failure behavior, security assumptions, and
configuration. Prefer names and tests over comments that restate syntax.

- Document public APIs and non-obvious contracts in the format used by the
  repository, such as YARD.
- Explain why for compatibility workarounds, unusual algorithms, security
  exceptions, and performance trade-offs.
- Update README, changelog, API documentation, or runbooks when behavior or
  operations change.
- Keep examples executable or clearly mark pseudocode.

```ruby
# DO: document the public contract and retry assumption.
# Returns the parsed response. Safe to retry after a timeout because the
# request is read-only.
def fetch_report(report_id)
  # ...
end
```

```ruby
# DO NOT: repeat the implementation in a comment.
count += 1 # increment count
```

Allowed exceptions: private, self-explanatory methods need no docstring;
generated API documentation may be canonical. Keep a comment when removing it
would make a security, compatibility, or concurrency decision easy to undo.

## Quick checklist

- [ ] Public behavior and operational assumptions are documented.
- [ ] Examples use current names and commands.
- [ ] Comments explain rationale and constraints, not mechanics.
- [ ] Documentation checks pass when configured.

## Sources and decisions

- [Ruby documentation](https://ruby-doc.org/), accessed 2026-09-30.
- [YARD documentation](https://yardoc.org/), accessed 2026-09-30.
- Repository API schemas and runbooks override generic documentation examples.

