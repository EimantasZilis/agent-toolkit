# JavaScript tests

## Apply this first

- Name tests by observable behavior and isolate setup, action, and assertions.
- Prefer deterministic unit tests; use integration tests for real module,
  network, filesystem, or browser boundaries.
- Assert useful outcomes and error behavior, not implementation details or
  incidental call counts.
- Restore mocks, timers, environment variables, and global state after each
  test; do not leave asynchronous work running.

## Quick checklist

- [ ] The test fails for the regression and passes for the fix.
- [ ] Tests cover success, relevant boundary values, and failure behavior.
- [ ] External boundaries are controlled or exercised intentionally.
- [ ] Test isolation and cleanup are explicit.

## DO / DO NOT

```js
test("returns an empty result when no records match", async () => {
  const result = await findRecords({ query: "missing" });

  expect(result).toEqual([]);
});
```

Do not assert private locals or snapshot large unstable objects when a focused
assertion expresses the contract more clearly.

## Allowed exceptions

An end-to-end test may use real services when its environment is provisioned,
isolated, and the test is explicitly categorized as integration/e2e. A snapshot
is acceptable for a stable, user-facing serialized contract with reviewable
diffs.

## Sources

- [MDN testing overview](https://developer.mozilla.org/en-US/docs/Learn/Tools_and_testing)
- [Node.js test runner](https://nodejs.org/api/test.html)
- [Testing Library guiding principles](https://testing-library.com/docs/guiding-principles)
