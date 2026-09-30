# TypeScript documentation

## Apply this first

- Document public types, functions, generic constraints, nullability, side
  effects, async behavior, thrown errors, and runtime validation when the type
  signature alone is insufficient.
- Prefer names, discriminated unions, and signatures that express the contract;
  comments should explain intent, invariants, or trade-offs.
- Keep JSDoc examples and generated API documentation synchronized with source.
- Avoid comments that claim compile-time guarantees the runtime does not enforce.

## Quick checklist

- [ ] Public API behavior and failure modes are clear.
- [ ] Runtime guarantees are distinguished from erased TypeScript types.
- [ ] Generic and overload behavior is documented where non-obvious.
- [ ] Examples compile or are tested when practical.

## DO / DO NOT

~~~ts
/**
 * Parses an external payload after validating its runtime shape.
 * @throws {ParseError} when the payload is not a supported user object.
 */
export function parseUser(payload: unknown): User {}
~~~

Do not document payload as a User merely because the implementation casts it;
the function's contract starts at the untrusted boundary.

## Allowed exceptions

Private, trivial helpers may need no comment when names and tests express the
contract. Generated API docs may use their generator's syntax when source
comments/types remain authoritative.

## Sources

- [TypeScript JSDoc reference](https://www.typescriptlang.org/docs/handbook/jsdoc-supported-types.html)
- [TypeScript declaration files](https://www.typescriptlang.org/docs/handbook/2/type-declarations.html)
- [TypeScript documentation](https://www.typescriptlang.org/docs/handbook/intro.html)
