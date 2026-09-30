# JavaScript linting and verification

## Apply this first

- Use the repository's configured formatter and ESLint configuration; do not
  invent competing style rules in a changed file.
- Run formatting in check mode where supported, then lint the changed paths,
  then run the relevant tests. Run the package's type checker when configured.
- Treat lint suppressions as code: scope them to the smallest line or block,
  state why they are safe, and remove stale suppressions.
- Keep lockfiles and generated outputs under their owning tool; do not hand
  edit them.
- Check dependencies, scripts, and configuration for secrets and unsafe
  shell or code injection before review completion.

## Quick checklist

- [ ] Repository formatter and linter pass for changed paths.
- [ ] Relevant tests and build/type checks pass.
- [ ] Suppressions are narrow, justified, and still necessary.
- [ ] No secrets or unrelated generated changes are included.

## DO / DO NOT

```js
// DO: explain a narrow, unavoidable suppression.
// eslint-disable-next-line no-await-in-loop -- requests must preserve order.
await send(item);

// DO NOT: disable an entire file to hide unrelated findings.
/* eslint-disable */
```

## Allowed exceptions

Generated code, vendored code, and compatibility shims may use an explicit
file-level override when the repository configuration identifies the path and
the generated/source owner is clear. Never weaken linting only to make a patch
pass.

## Sources

- [ESLint rules reference](https://eslint.org/docs/latest/rules/)
- [ESLint configuration](https://eslint.org/docs/latest/use/configure/)
- [Prettier rationale](https://prettier.io/docs/rationale.html)
- [Prettier option philosophy](https://prettier.io/docs/option-philosophy.html)
