# Ruby core

## Apply this first

Write idiomatic, unsurprising Ruby. Keep methods and objects focused, make
names express domain intent, prefer simple control flow, and make mutation and
side effects visible. Match the repository's supported Ruby version before
using newer syntax.

- Prefer small public methods that delegate cohesive work to private methods or
  domain objects.
- Use keyword arguments for options and immutable values when mutation is not
  required.
- Keep `nil` handling explicit; use safe navigation only when `nil` is valid
  and the resulting `nil` is handled.
- Avoid metaprogramming when ordinary methods, modules, or composition are
  clearer.
- Keep dependencies and public APIs narrow; make side effects visible at the
  boundary that owns them.

```ruby
# DO: make the operation explicit and keep the boundary narrow.
def publish_article(article, actor:)
  ArticlePublisher.new(article, actor:).call
end

# DO NOT: hide writes in a generic reflective helper.
def process(record, action, **options)
  record.public_send(action, **options)
end
```

Allowed exceptions: use metaprogramming or a DSL when it is the documented
extension point of a dependency or an established local convention; explain
the invariant in a comment or test when the behavior is not obvious.

## Quick checklist

- [ ] Names, method boundaries, and mutation are clear.
- [ ] Syntax is supported by the project's Ruby version.
- [ ] `nil`, truthiness, and empty-collection behavior are intentional.
- [ ] The diff avoids unrelated refactoring and clever indirection.

## Sources and decisions

- [Ruby syntax and semantics documentation](https://ruby-doc.org/docs/ruby-doc-bundle/UsersGuide/rg/), accessed 2026-09-30.
- [RuboCop overview and style-guide relationship](https://docs.rubocop.org/rubocop/), accessed 2026-09-30.
- Exact style preferences remain repository-local because RuboCop supports
  configurable conventions across Ruby projects.

