# Python linting and implementation guidance

## Apply this first

- Follow nearby module conventions and keep unrelated lines unchanged.
- Use descriptive `snake_case` names for functions and variables and
  `PascalCase` for classes.
- Type every input and output; avoid `Any`.
- Give each function one main responsibility and normally keep it below 40
  lines.
- Prefer keyword arguments, early returns, and readable vertical spacing.
- Use blank lines to separate related variable groups and distinct logical
  phases inside functions; do not mechanically separate every statement.
- Keep imports grouped as standard library, third party, then local modules.
- Do not use wildcard imports or bare exception handlers.
- Preserve the original cause when converting an exception.

## Quick checklist

- [ ] Names use the repository's `snake_case` / `PascalCase` conventions.
- [ ] Inputs and return values are typed.
- [ ] Mutable defaults and hidden side effects are absent.
- [ ] `None` comparisons use `is` or `is not`.
- [ ] Comprehensions remain readable.
- [ ] Imports, definitions, and logical phases have clear spacing.
- [ ] Related variables and distinct function phases have clear vertical
      spacing after the formatter pass.
- [ ] Helpers are public unless a real API boundary justifies `_`.
- [ ] Functions remain focused and normally fit within 40 lines.

## Baseline design rules

Match existing patterns, avoid unrelated refactors, and prefer composition over
inheritance. Prefer a cohesive class when several operations target one
application, domain, or infrastructure responsibility, especially when the
operations share stable configuration, collaborators, or lifecycle state.
Inject those collaborators through the constructor when it improves reuse and
testability, while keeping per-operation mutable state local to each method.
Do not wrap unrelated pure functions in a class merely to satisfy this
preference.

Group class methods around one focused responsibility and use small reusable
units for distinct validation, transformation, persistence, querying, and
orchestration phases. A public method should read as a clear workflow over
those units; split it when it mixes responsibilities or has several meaningful
phases, but do not mechanically split a short linear method. Use a
private-looking helper only when callers, subclasses, or neighboring modules
should be prevented from depending on it; brevity alone is not enough.
When refactoring existing code, introduce these boundaries incrementally at
cohesive seams and preserve behavior and public contracts unless the task
explicitly changes them.

### ✅ DO

```python
from typing import Iterable


def index_contact_addresses(
    contacts: Iterable["Contact"],
) -> dict[int, str]:
    address_by_id: dict[int, str] = {}
    for contact in contacts:
        if not contact.address:
            continue

        address_by_id[contact.id] = contact.address

    return address_by_id
```

### ❌ DO NOT

```python
def make_map(items: list) -> dict:
    return {item.id: item.email for item in items}
```

The second version uses vague names, untyped values, and does not make the
missing-address behavior explicit.

## Function size and responsibility

Split a function when it mixes orchestration with aggregation, transformation,
validation, persistence, or external effects. A short linear function need not
be split merely to satisfy a number. If a cohesive function must exceed 40
lines, document why the boundary is difficult to improve and do not suppress
complexity checks without recording that reason.

An orchestration wrapper may coordinate work, but independently understandable
calculation, transformation, and persistence belong in named helpers.

## Spacing and imports

Use two blank lines between top-level definitions and one blank line between
distinct phases inside a function. Keep imports separate from definitions and
ordered stdlib -> third party -> local. Preserve surrounding formatting when
editing an existing file.

## Context-manager form

When a context manager can also decorate a function and the whole function
needs the managed scope, use the decorator form:

```python
@managed_context
def consume_messages() -> None:
    ...
```

Use an inline `with managed_context():` block when only part of the function
needs the scope, when nesting or lifetime matters, or when another concrete
design constraint requires it.

### Allowed exception

An inline block is preferred whenever the managed lifetime is intentionally
shorter than the function or must be nested. This is a scope decision, not a
style violation.

## Naming and annotations

Use `snake_case` for functions and variables, `PascalCase` for classes, and
avoid cryptic one- or two-letter names except for trivial loop variables.
Annotate all inputs and outputs, use `-> None` for procedures, and avoid
`Any`.
