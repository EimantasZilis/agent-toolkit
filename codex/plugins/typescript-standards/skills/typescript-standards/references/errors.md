# TypeScript errors and asynchronous control flow

## Apply this first

- Throw Error instances or domain subclasses with safe, actionable context;
  type the error boundary as unknown and narrow it before inspection.
- Catch only to recover, translate, add context, or clean up. Preserve causes
  when translating and do not use as to pretend arbitrary values are errors.
- Return or await Promises that callers must observe. Define cancellation,
  timeout, retry, and idempotency behavior at external boundaries.
- Use discriminated result types when expected failure is part of normal control
  flow; reserve exceptions for exceptional failure.

## Quick checklist

- [ ] Error paths remain observable and preserve safe context and cause.
- [ ] unknown catch values are narrowed safely.
- [ ] Promise rejection and cleanup behavior is tested.
- [ ] Retries are bounded and do not repeat unsafe side effects.

## DO / DO NOT

~~~ts
try {
  return await readConfig(key);
} catch (error: unknown) {
  throw new ConfigReadError("Could not read " + key, { cause: error });
}
~~~

Do not catch unknown and access message without narrowing, or create a
floating Promise whose rejection no caller can observe.

## Allowed exceptions

Best-effort telemetry may be isolated, bounded, and intentionally ignored.
Cleanup may aggregate its error with the primary failure. A framework callback
may use a void Promise expression when the framework explicitly owns its
rejection handling and the contract is documented.

## Sources

- [TypeScript useUnknownInCatchVariables](https://www.typescriptlang.org/tsconfig/useUnknownInCatchVariables.html)
- [MDN async function](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Statements/async_function)
- [MDN Error cause](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Error/cause)
