# TypeScript modules and project configuration

## Apply this first

- Match module, moduleResolution, package type, file extensions, and
  bundler/runtime behavior. A tsconfig is a deployment contract, not just an
  editor preference.
- Use project references or separate configs when server, browser, worker,
  test, and shared environments have different globals or module behavior.
- Keep public exports intentional and declaration output compatible with the
  supported consumers.
- Prefer verbatimModuleSyntax or the repository's equivalent when it makes
  type-only imports and emitted runtime imports unambiguous.

## Quick checklist

- [ ] The emitted module format runs in every supported target environment.
- [ ] Type-only imports do not become runtime dependencies.
- [ ] tsconfig scope excludes generated or unrelated environments.
- [ ] Public export and declaration changes have consumer coverage.

## DO / DO NOT

~~~ts
// DO: make a type-only dependency explicit.
import type { User } from "./user.js";
import { loadUser } from "./user.js";

// DO NOT: rely on compiler settings to guess whether this import is runtime code.
import { User } from "./user.js";
~~~

## Allowed exceptions

Bundlers may intentionally use moduleResolution bundler and extensionless
specifiers; Node-targeted packages may need nodenext and explicit extensions.
Follow the official configuration for that target and test emitted code rather
than applying one module policy everywhere.

## Sources

- [TypeScript modules introduction](https://www.typescriptlang.org/docs/handbook/modules.html)
- [Choosing TypeScript module options](https://www.typescriptlang.org/docs/handbook/modules/guides/choosing-compiler-options.html)
- [TypeScript verbatimModuleSyntax](https://www.typescriptlang.org/tsconfig/verbatimModuleSyntax.html)
- [Node.js packages](https://nodejs.org/api/packages.html)
