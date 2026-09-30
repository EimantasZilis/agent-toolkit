# Ruby security

## Apply this first

Treat external input, files, environment values, serialized data, and command
arguments as untrusted. Validate at boundaries and make dangerous operations
explicit.

- Keep credentials in the configured secret store or environment and redact
  them from errors, logs, fixtures, and test output.
- Prefer parameterized APIs and safe serialization; never interpolate untrusted
  data into SQL, shell commands, HTML, or file paths.
- Validate file type, size, path, and permissions before reading or writing.
- Use allowlists for commands, algorithms, protocols, and deserialization types.
- Keep dependency changes reviewed and constrained by the lockfile.

```ruby
# DO: pass arguments as data to a constrained API.
Open3.capture3("git", "show", revision)
```

```ruby
# DO NOT: interpolate untrusted data into a shell command.
system("git show #{revision}")
```

Allowed exceptions: a raw command or parser may be necessary at a system
boundary; constrain its inputs, document the trust model, and add regression
tests for rejected values. Test credentials may use fake values only.

## Quick checklist

- [ ] Inputs and deserialization boundaries are validated.
- [ ] Secrets are absent from source, logs, fixtures, and test output.
- [ ] Shell, filesystem, SQL, HTML, and network calls use safe APIs.
- [ ] Dependencies and lockfile changes are intentional.
- [ ] Security-sensitive behavior has a regression test.

## Sources and decisions

- [Ruby security documentation](https://www.ruby-lang.org/en/documentation/quickstart/), accessed 2026-09-30.
- [OWASP injection prevention guidance](https://owasp.org/www-community/attacks/Command_Injection), accessed 2026-09-30.
- The generic pack covers Ruby boundaries; framework-specific protections belong
  in an extension such as `ruby-rails-standards`.

