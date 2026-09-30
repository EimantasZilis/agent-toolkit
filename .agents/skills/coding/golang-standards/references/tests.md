# Go tests

## Apply this first

- Put tests in `_test.go` files with `TestXxx`, `BenchmarkXxx`,
  `ExampleXxx`, or `FuzzXxx` entry points accepted by `go test`.
- Test observable behavior and failure contracts. Prefer table-driven tests
  when cases share setup, and use subtest names that identify the scenario.
- Keep tests deterministic and isolated. Reset mutable global state, avoid
  sleeps for synchronization, and use real integration boundaries only when
  their environment is explicit and controlled.
- Assert errors and important output precisely enough to catch regressions;
  use `errors.Is` or `errors.As` for wrapped error contracts.
- Run race-enabled tests for concurrent code and seed fuzz tests with known
  boundary cases before spending time on fuzzing.

~~~go
func TestParse(t *testing.T) {
    tests := []struct{ name, input string; want int }{
        {name: "empty", input: "", want: 0},
        {name: "number", input: "42", want: 42},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Parse(tt.input)
            if got != tt.want { t.Fatalf("Parse(%q) = %d, want %d", tt.input, got, tt.want) }
        })
    }
}
~~~

## Quick checklist

- [ ] The test fails for the regression before the fix and passes after it.
- [ ] Success, boundary, and failure behavior are covered where relevant.
- [ ] Tests do not depend on order, wall-clock timing, or leaked global state.
- [ ] `-race`, fuzz, integration, or benchmark coverage is used when relevant.
- [ ] Test names and failure messages identify the scenario and values.

## DO / DO NOT

~~~go
// DO: capture the loop variable for parallel subtests when the repository's
// minimum Go version makes that necessary.
for _, tt := range tests {
    tt := tt
    t.Run(tt.name, func(t *testing.T) { t.Parallel(); testCase(t, tt) })
}

// DO NOT: synchronize a test with an arbitrary sleep.
time.Sleep(time.Second)
~~~

## Allowed exceptions

Use a small sleep only when testing a time-based contract and pair it with a
bounded timeout. Avoid `t.Parallel` when the test touches shared process
state, and use external services only in explicitly tagged or separately
configured integration tests.

## Sources

- [testing package](https://pkg.go.dev/testing) (accessed 2026-09-30)
- [Add a test](https://go.dev/doc/tutorial/add-a-test) (accessed 2026-09-30)
- [Go fuzzing tutorial](https://go.dev/doc/tutorial/fuzz) (accessed 2026-09-30)
