# Ruby linting

## Apply this first

Treat checked-in formatter and linter configuration as executable project
policy. Run tools through Bundler so the lockfile determines their versions.

- Run `bundle exec rubocop` or the repository's documented equivalent on
  changed Ruby paths.
- Read applicable cop configuration before disabling a cop; prefer a narrow,
  explained disable around exceptional code.
- Treat unsafe autocorrection as a reviewable code change and inspect the diff.
- Check for generated files, secrets, accidental debug output, and dependency
  changes before finalizing.
- Run the repository's full verification command when one exists.

```yaml
# DO: keep project style in the checked-in configuration.
AllCops:
  TargetRubyVersion: 3.3
```

```ruby
# DO NOT: suppress a rule globally to make one legacy call pass.
# rubocop:disable Metrics/MethodLength
```

Allowed exceptions: generated, vendored, compatibility, or deliberately
supported legacy code may use a scoped disable; state the reason and, where
useful, a removal condition. Never weaken unrelated checks to hide a failure.

## Quick checklist

- [ ] Changed Ruby paths were formatted and linted with project-scoped tools.
- [ ] The configured Ruby target matches the supported runtime.
- [ ] Unsafe autocorrections were reviewed manually.
- [ ] Disables are narrow and justified.
- [ ] Relevant tests and repository verification pass.

## Sources and decisions

- [RuboCop configuration](https://docs.rubocop.org/rubocop/latest/configuration.html), accessed 2026-09-30.
- [RuboCop style cops](https://docs.rubocop.org/rubocop/latest/cops_style.html), accessed 2026-09-30.
- RuboCop is the executable baseline, while the repository chooses the exact
  enabled cops and supported Ruby version.

