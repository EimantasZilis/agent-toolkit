# Rails core overlay

## Apply this first

Apply [ruby-standards core](../../ruby-standards/references/core.md) first.
This overlay covers the Rails-specific consequences of those Ruby rules.

Keep framework boundaries legible. Match Rails naming and file/constant
alignment, make lifecycle and side effects visible, and prefer documented
framework extension points over clever framework bypasses.

- Prefer small public methods that delegate cohesive work to private methods or
  domain objects.
- Use keyword arguments for options and immutable values when mutation is not
  required.
- Keep `nil` handling explicit; use safe navigation only when `nil` is a valid
  state and the resulting `nil` is handled.
- Keep constant and file naming aligned so Zeitwerk can autoload and reload it.
- Keep controllers, models, jobs, views, and service objects focused on their
  framework boundary; do not hide workflows in broad callbacks.
- Prefer Rails APIs and the application's established extension points before
  adding a new abstraction.

```ruby
# DO: make the Rails boundary explicit and keep it narrow.
def publish_article(article, actor:)
  ArticlePublisher.new(article, actor:).call
end

# DO NOT: hide writes, authorization, and notifications in a broad callback.
before_action :prepare_everything
```

Allowed exceptions: use Rails metaprogramming or a DSL when it is a documented
framework extension point or an established local convention; explain the
invariant in a comment or test when the behavior is not obvious.

## Quick checklist

- [ ] The Ruby baseline was applied before this Rails overlay.
- [ ] File paths, constants, and framework boundaries agree.
- [ ] Lifecycle hooks and side effects are intentional and testable.
- [ ] The diff does not introduce unrelated Rails refactoring.

## Sources and decisions

- [Ruby syntax and semantics documentation](https://ruby-doc.org/docs/ruby-doc-bundle/UsersGuide/rg/), accessed 2026-09-30.
- [RuboCop overview and style-guide relationship](https://docs.rubocop.org/rubocop/), accessed 2026-09-30.
- RuboCop and the community Ruby Style Guide agree on consistent, automatable
  style; this pack intentionally leaves exact line length and cop choices to
  the repository configuration.
