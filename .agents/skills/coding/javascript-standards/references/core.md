# JavaScript core

## Apply this first

- Prefer small, named functions with one clear responsibility and explicit
  inputs and outputs.
- Use `const` by default; use `let` only when reassignment is required. Avoid
  implicit globals and mutation of shared state.
- Prefer strict equality and explicit branching. Handle `null` and
  `undefined` intentionally rather than relying on truthiness for domain data.
- Keep side effects at boundaries. Make asynchronous work explicit with
  `async`/`await` or a returned Promise, and always settle or propagate it.
- Prefer modules over scripts and avoid clever coercion; make conversions
  visible at the boundary.

```js
export async function loadUser(id, fetchUser) {
  const user = await fetchUser(id);
  if (user == null) return undefined;
  return { id: user.id, name: String(user.name) };
}
```

## Quick checklist

- [ ] No accidental globals, unnecessary mutation, or hidden asynchronous work.
- [ ] Branches handle the domain's missing, empty, and invalid values.
- [ ] Functions and module exports have focused responsibilities.
- [ ] Changed behavior has a relevant test or a recorded reason it cannot be tested.

## DO / DO NOT

```js
// DO: make the fallback explicit.
const timeoutMs = options.timeoutMs ?? DEFAULT_TIMEOUT_MS;

// DO NOT: turn valid falsy values into a fallback.
const timeoutMs = options.timeoutMs || DEFAULT_TIMEOUT_MS;
```

## Allowed exceptions

Use `||` when every falsy value is intentionally equivalent to missing. Use
mutation for a measured hot path or a stateful API only when ownership and
invariants are clear in the surrounding code.

## Sources

- [ECMAScript language specification](https://tc39.es/ecma262/)
- [MDN JavaScript Guide](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide)
- [MDN strict equality](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/Strict_equality)
