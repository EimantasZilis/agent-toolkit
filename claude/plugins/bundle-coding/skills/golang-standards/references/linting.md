# Go linting and verification

## Apply this first

- Use the repository-pinned Go version and configured tools. Run `gofmt` on
  changed files, `go vet ./...`, the configured linter, and relevant tests.
- Keep `go.mod` and `go.sum` consistent with the intended module graph. Review
  dependency additions, licenses, generated files, and build-tag changes.
- Treat compiler, vet, race-detector, and linter findings as defects unless a
  narrow suppression is justified next to the affected code or in checked-in
  configuration.
- Run `go test ./...` or the repository's narrower command before completion;
  use `go test -race` for code that shares mutable state.
- When dependencies or externally reachable code changes, use the repository's
  vulnerability workflow, such as `govulncheck`, if configured.

~~~sh
gofmt -w ./path/to/changed.go
go vet ./...
go test ./...
~~~

## Quick checklist

- [ ] Changed files are formatted and imports are clean.
- [ ] Vet, configured lint, relevant tests, and repository verification pass.
- [ ] Dependency, module, build-tag, and generated-output changes are intentional.
- [ ] Suppressions name the rule and the concrete reason.
- [ ] Security scanning ran when the repository requires it.

## DO / DO NOT

~~~go
// DO: narrow a suppression and explain the boundary.
//nolint:gochecknoglobals // This immutable registry is required by the plugin API.
var registry = map[string]Factory{}

// DO NOT: disable a whole class of checks to make a change pass.
//nolint:all
~~~

## Allowed exceptions

Generated code, vendored code, platform-specific files, and compatibility
shims may use repository-configured exclusions when the generator, owner, or
supported platform is clear and the excluded path is still checked where
practical. Never hide a finding merely because fixing it is inconvenient.

## Sources

- [gofmt documentation](https://pkg.go.dev/cmd/gofmt) (accessed 2026-09-30)
- [go vet documentation](https://pkg.go.dev/cmd/vet) (accessed 2026-09-30)
- [Go modules reference](https://go.dev/ref/mod) (accessed 2026-09-30)
- [govulncheck](https://go.dev/doc/security/vuln/) (accessed 2026-09-30)
