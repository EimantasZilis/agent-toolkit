# JavaScript documentation

## Apply this first

- Document public modules, functions, configuration, side effects, thrown
  errors, async behavior, and examples when they are not self-evident.
- Prefer comments that explain intent, invariants, or trade-offs; let names and
  types express mechanics.
- Keep JSDoc synchronized with runtime behavior and validate examples when
  they are executable documentation.

## Quick checklist

- [ ] Public behavior and failure modes are documented.
- [ ] Comments explain why, not a restatement of the code.
- [ ] Examples use the supported module/runtime contract.
- [ ] Documentation and JSDoc match the changed implementation.

## DO / DO NOT

```js
/**
 * Reads a config value, returning `undefined` when the key is absent.
 * @param {string} key
 * @returns {Promise<string | undefined>}
 * @throws {ConfigReadError} when the backing store is unavailable.
 */
export async function readConfig(key) {}
```

Do not add a long comment that merely narrates an obvious assignment.

## Allowed exceptions

Private, trivial helpers may need no JSDoc when names, tests, and local context
fully express their contract. Generated API documentation may follow its
generator's syntax when the source remains authoritative.

## Sources

- [JSDoc documentation](https://jsdoc.app/)
- [MDN JavaScript Guide](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide)
