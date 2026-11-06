# Bash tests

## Apply this first

- Test observable behavior: exit status, stdout, stderr, filesystem changes,
  environment changes, and invoked commands where those are part of the API.
- Use a shell test framework when the repository has enough command-oriented
  integration tests to justify one; otherwise, small focused shell assertions
  are sufficient. Use isolated temporary directories and deterministic
  fixtures.
- Give each test one reason to fail. Name tests with the user-visible behavior
  and cover success, invalid input, missing dependencies, and failure paths.
- Capture command status and output explicitly when asserting command behavior,
  including pipeline status where pipelines are part of the interface.
- Keep setup and teardown explicit. Make teardown safe after partial setup and
  do not allow test data to escape the temporary directory.
- Run tests with a controlled `PATH`, locale, working directory, and relevant
  environment so the suite does not accidentally depend on the developer shell.

## Quick checklist

- [ ] Tests assert both status and relevant output or side effects.
- [ ] Success and meaningful failure paths are covered.
- [ ] Tests use isolated, disposable fixtures and clean them up.
- [ ] External commands and environment assumptions are controlled.
- [ ] Pipeline assertions test the intended pipeline status.
- [ ] The full relevant test command passes, not only an individual case.

## DO

```bash
test_deploy_rejects_missing_configuration() {
    local output status
    set +e
    output="$(deploy --config "$TEST_TMPDIR/missing.conf" 2>&1)"
    status=$?
    set -e

    [[ "$status" -eq 1 ]]
    [[ "$output" == *"configuration file not found"* ]]
}
```

## DO NOT

```bash
@test "deploy works" {
    deploy --config config.conf
    # No status, output, or side-effect assertion.
}
```

## Concrete exceptions

- A small pure helper may be tested directly with Bash assertions when the
  repository has no test framework; keep the test command repeatable and
  document the convention.
- Snapshot or exact-output assertions are appropriate for a stable CLI output
  contract; otherwise assert stable substrings and status instead of incidental
  formatting.
- A test may use a real external dependency when the behavior under test is the
  integration itself; mark it as integration-only and provide a deterministic
  local or CI prerequisite.

## Source traceability

- Testable command behavior and return values:
  [Google Shell Style Guide](https://google.github.io/styleguide/shellguide.html) (accessed 2026-09-30).
