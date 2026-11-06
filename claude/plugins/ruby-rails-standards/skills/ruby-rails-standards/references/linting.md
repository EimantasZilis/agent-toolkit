# Rails linting overlay

## Apply this first

Apply [ruby-standards linting](../../ruby-standards/references/linting.md)
first. Treat the checked-in formatter and linter configuration as executable
project policy, then add Rails cops only when the application uses them.

- Run `bundle exec rubocop` or the repository's documented equivalent on
  changed Ruby paths with the repository's configured Rails extension.
- Read the applicable cop configuration before disabling a cop. Prefer a
  narrow, explained disable around the exceptional code.
- Treat unsafe autocorrection as a reviewable code change, not a mechanical
  cleanup. Inspect the resulting diff and tests.
- Check for secrets, generated files, migrations, and accidental debug output
  before finalizing.
- Run the repository's full verification command when one exists.

```yaml
# DO: use the project's Rails plugin and target version.
plugins:
  - rubocop-rails
AllCops:
  TargetRailsVersion: 8.0
```

```ruby
# DO NOT: suppress a rule globally just to make a single legacy call pass.
# rubocop:disable Metrics/MethodLength
```

Allowed exceptions: a generated file, compatibility shim, vendored code, or
deliberately supported legacy boundary may use a scoped disable; state the
reason and, where useful, a removal condition. Never change lint configuration
merely to hide an unrelated failure.

## Quick checklist

- [ ] The Ruby linting baseline was applied.
- [ ] Changed Rails paths were formatted and linted with project-scoped tools.
- [ ] Rails cops target the application's actual Rails version.
- [ ] Unsafe autocorrections were reviewed manually.
- [ ] Disables are narrow, justified, and do not weaken unrelated code.
- [ ] Relevant tests and repository verification pass.

## Sources and decisions

- [RuboCop configuration](https://docs.rubocop.org/rubocop/latest/configuration.html), accessed 2026-09-30.
- [RuboCop Rails usage and target version](https://docs.rubocop.org/rubocop-rails/usage.html), accessed 2026-09-30.
- [RuboCop Rails cops](https://docs.rubocop.org/rubocop-rails/latest/cops.html), accessed 2026-09-30.
- The official RuboCop docs establish configuration and plugin behavior; exact
  enabled cops remain repository-local because applications intentionally vary.
