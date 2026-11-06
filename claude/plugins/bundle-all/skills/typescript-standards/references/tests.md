# TypeScript tests

## Apply this first

- Test observable behavior and type/runtime boundaries, not private
  implementation details.
- Keep setup, action, and assertions distinct. Use compile-time assertions
  when a public type contract is part of the behavior, alongside runtime tests.
- Prefer deterministic unit tests; use integration tests for real module,
  filesystem, network, browser, or serialization boundaries.
- Restore mocks, timers, environment, and global state; ensure every async
  operation is awaited and test isolation is preserved.

## Quick checklist

- [ ] The regression test fails before the fix and passes after it.
- [ ] Runtime and, where relevant, compile-time contracts are covered.
- [ ] Boundary values and failure behavior are asserted.
- [ ] Async work and test doubles are cleaned up.

## DO / DO NOT

~~~ts
it("rejects an invalid user payload", async () => {
  await expect(parseUser({ id: 4 })).rejects.toThrow("id");
});
~~~

Do not make a test pass by casting its fixture to the target type; construct a
valid fixture or intentionally test the invalid boundary.

## Allowed exceptions

An end-to-end test may use real services when its environment is isolated and
the test is explicitly categorized. A type-level test may contain no runtime
assertion when its sole contract is assignability, but it must be run by the
repository's type-test command.

## Sources

- [TypeScript type compatibility](https://www.typescriptlang.org/docs/handbook/type-compatibility.html)
- [Node.js test runner](https://nodejs.org/api/test.html)
- [Testing Library guiding principles](https://testing-library.com/docs/guiding-principles)
