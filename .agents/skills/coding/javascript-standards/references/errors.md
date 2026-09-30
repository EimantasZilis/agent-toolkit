# JavaScript errors and asynchronous control flow

## Apply this first

- Throw `Error` instances (or domain subclasses) with actionable context; do
  not throw strings or silently catch failures.
- Catch only when the layer can recover, translate, add context, or clean up.
  Otherwise let the rejection/error propagate.
- Preserve the original cause when translating errors (`new Error(message,
  { cause })` where supported).
- Await or return every Promise that the caller must observe. Define timeout,
  cancellation, retry, and idempotency behavior at the boundary.

## Quick checklist

- [ ] Failure paths are observable and include safe diagnostic context.
- [ ] Catches have a recovery, translation, or cleanup purpose.
- [ ] Promise rejection and resource cleanup behavior is tested.
- [ ] Retries are bounded and do not duplicate non-idempotent work.

## DO / DO NOT

```js
// DO: translate while preserving the cause.
try {
  return await client.read(key);
} catch (error) {
  throw new ConfigReadError(`Could not read ${key}`, { cause: error });
}

// DO NOT: hide a rejected operation.
client.read(key).catch(() => undefined);
```

## Allowed exceptions

Intentionally best-effort telemetry may ignore a failure only when it is
isolated from the primary operation, bounded, and documented. Cleanup errors
may be aggregated with the primary error rather than replacing it.

## Sources

- [MDN async function](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Statements/async_function)
- [MDN Promise rejection handling](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide/Using_promises)
- [MDN Error cause](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Error/cause)
