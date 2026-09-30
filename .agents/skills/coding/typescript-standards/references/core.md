# TypeScript core

## Apply this first

- Enable and preserve strict checking unless the repository documents a
  narrower compatibility constraint.
- Prefer precise domain types, discriminated unions, and narrowing over
  assertions. Model invalid states so they are difficult to construct.
- Use unknown at untrusted boundaries and validate it before use. Avoid any;
  if unavoidable, isolate it and document the boundary.
- Prefer const, immutable data flow, and small functions with explicit return
  types at public or complex boundaries.
- Keep runtime behavior in mind: types are erased and cannot validate input,
  permissions, serialization, or external responses.

~~~ts
type Result = { ok: true; value: string } | { ok: false; message: string };

function messageOf(result: Result): string {
  return result.ok ? result.value : result.message;
}
~~~

## Quick checklist

- [ ] New types describe the domain rather than merely mirroring implementation.
- [ ] Untrusted values are validated at runtime.
- [ ] No new broad assertion or any hides a type error.
- [ ] Public/complex functions expose an intentional return type.

## DO / DO NOT

~~~ts
// DO: narrow an unknown value.
function isUser(value: unknown): value is { id: string } {
  return typeof value === "object" && value !== null &&
    "id" in value && typeof value.id === "string";
}

// DO NOT: assert data into existence.
const user = payload as User;
~~~

## Allowed exceptions

Use an assertion at a proven boundary such as a framework registration hook or
validated schema result when the proof is local, stable, and documented. Use
any only for an unavoidable third-party or migration boundary, behind a small
adapter with tests.

## Sources

- [TypeScript Everyday Types](https://www.typescriptlang.org/docs/handbook/2/everyday-types.html)
- [TypeScript narrowing](https://www.typescriptlang.org/docs/handbook/2/narrowing.html)
- [TypeScript strict compiler option](https://www.typescriptlang.org/tsconfig/strict.html)
