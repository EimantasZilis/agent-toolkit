# TypeScript linting and verification

## Apply this first

- Use the repository's configured formatter, ESLint flat/config setup, and
  TypeScript version; do not add competing style rules in source files.
- Run formatting checks, lint changed paths, run tsc --noEmit or the package's
  declared type-check command, then run relevant tests and builds.
- Keep compiler strictness and declaration settings intentional; do not silence
  errors with skipLibCheck, any, or broad disables without a recorded reason.
- Scope lint suppressions narrowly and explain why the rule is inapplicable.
- Review package scripts, dependencies, generated declarations, and config for
  secrets or unsafe evaluation before completion.

## Quick checklist

- [ ] Formatter, linter, type checker, and relevant tests pass.
- [ ] Compiler-option changes are justified and do not weaken safety silently.
- [ ] Suppressions are narrow, justified, and still necessary.
- [ ] Generated output and lockfile changes are tool-owned and intentional.

## DO / DO NOT

~~~ts
// DO: isolate and justify a boundary suppression.
// eslint-disable-next-line @typescript-eslint/no-unsafe-argument -- SDK accepts its validated payload shape.
sdk.send(validatedPayload);

// DO NOT: disable type safety project-wide to bypass one incompatibility.
// "strict": false
~~~

## Allowed exceptions

Generated declarations, vendored code, and compatibility shims may use a
repository-configured override when the source owner and regeneration command
are clear. A library may choose a less strict public declaration boundary for
consumer compatibility, but its implementation should remain checked.

## Sources

- [TypeScript compiler options](https://www.typescriptlang.org/tsconfig/)
- [TypeScript strict](https://www.typescriptlang.org/tsconfig/strict.html)
- [ESLint TypeScript project](https://typescript-eslint.io/)
- [Prettier TypeScript](https://prettier.io/docs/options.html)
