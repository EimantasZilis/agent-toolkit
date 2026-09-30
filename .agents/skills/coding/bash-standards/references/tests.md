# Bash tests

## Apply this first

- Test observable behavior: exit status, stdout, stderr, filesystem changes,
  environment changes, and invoked commands where those are part of the API.
- Prefer a test framework such as Bats for command-oriented integration tests;
  use isolated temporary directories and deterministic fixtures.
- Give each test one reason to fail. Name tests with the user-visible behavior
  and cover success, invalid input, missing dependencies, and failure paths.
- Use Bats `run` when assertions need a command's status or output. For a
  pipeline, use `bats_pipe` or test the pipeline through a wrapper command.
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
@test "deploy rejects a missing configuration" {
    run -1 deploy --config "$BATS_TEST_TMPDIR/missing.conf"

    [ "$status" -eq 1 ]
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

- A small pure helper may be tested directly with Bash assertions instead of
  Bats when the repository has no test framework; keep the test command
  repeatable and document the convention.
- Snapshot or exact-output assertions are appropriate for a stable CLI output
  contract; otherwise assert stable substrings and status instead of incidental
  formatting.
- A test may use a real external dependency when the behavior under test is the
  integration itself; mark it as integration-only and provide a deterministic
  local or CI prerequisite.

## Source traceability

- Bats command execution, status/output variables, fixtures, hooks, and
  pipeline handling: [Bats writing tests](https://bats-core.readthedocs.io/en/stable/writing-tests.html) (accessed 2026-09-30).
- Testable command behavior and return values:
  [Google Shell Style Guide](https://google.github.io/styleguide/shellguide.html) (accessed 2026-09-30).

Bats documents that `run` captures status and output and that ordinary pipes are
parsed outside `run`; this pack makes both points explicit to prevent tests that
appear to cover a pipeline but do not.
