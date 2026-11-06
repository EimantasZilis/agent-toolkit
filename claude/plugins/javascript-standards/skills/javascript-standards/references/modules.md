# JavaScript modules

## Apply this first

- Follow the package's declared module system (`"type"`, `.mjs`, `.cjs`, or
  the repository's bundler), and do not mix systems accidentally.
- Prefer named exports for stable APIs and keep import paths explicit and
  portable for the target runtime.
- Keep module initialization free of surprising I/O and hidden global state.
- Treat package `exports`, dependency versions, and public entry points as API
  surface; update consumers and tests when they change.

## Quick checklist

- [ ] The changed file is interpreted by the intended runtime and toolchain.
- [ ] Imports/exports follow local ordering and boundary conventions.
- [ ] Public entry-point changes have consumer or integration coverage.
- [ ] No circular dependency or import-time side effect was introduced.

## DO / DO NOT

```js
// DO: use the package's declared ESM contract.
import { parseConfig } from "./config.js";

// DO NOT: rely on a directory or extension guess that only one tool resolves.
import parseConfig from "./config";
```

## Allowed exceptions

Use extensionless imports or CommonJS interop when the repository's bundler or
runtime explicitly requires it. Keep that rule at the package boundary and
test the emitted/runtime form.

## Sources

- [MDN JavaScript modules](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide/Modules)
- [Node.js packages](https://nodejs.org/api/packages.html)
- [ECMAScript modules](https://tc39.es/ecma262/#sec-modules)
