# Docstrings & Comments

## Apply this first

- Add/update docstrings for new or changed behavior.
- Keep docstrings behavior-accurate; do not drift from code.
- Prefer concise comments that explain why, not what.
- Avoid planning/process notes in code comments.
- In Django context, use model-oriented wording (for example: "record").
- Do not do style-only docstring rewrites.

## Quick checklist

- [ ] New/changed behavior has updated docstrings.
- [ ] Docstrings match behavior and exception semantics.
- [ ] One-line docs are imperative and end with a period.
- [ ] Multi-line docs use summary + blank line + details.
- [ ] Comments explain why, not obvious what.

---

## Core rules

- Required for new or changed code
- Update when behavior changes
- Must reflect actual behavior
- Do not rewrite only for style
- No planning/workflow references

---

## Function docstrings

### ✅ DO
```python
def acquire_lock(resource_id: int) -> None:
    """Create a lock record to prevent duplicate processing.

    Raises:
        IntegrityError:
            If lock already exists
    """
    Lock.objects.create(resource_id=resource_id)
```

### ❌ DO NOT
```python
def acquire_lock(resource_id):
    # create lock
    Lock.objects.create(resource_id=resource_id)
```

---

## Django wording

- Use "record" not "row"
- Use model names (`Order`), not SQL terms

### ✅ DO
```python
"""Return records for `Order`."""
```

### ❌ DO NOT
```python
"""Return rows from table."""
```

---

## Comments

- Explain WHY, not WHAT

### ✅ DO
```python
# Prevent duplicate processing across concurrent requests
```

### ❌ DO NOT
```python
# loop through items
```

---

## Formatting and structure (PEP 257 aligned)

- Use triple double quotes (`"""..."""`) for docstrings.
- One-line docstrings should be a single imperative sentence ending with a period.
- Multi-line docstrings should include:
  - a summary line
  - a blank line
  - additional detail only when needed
- Prefer documenting behavior, side effects, raised exceptions, and constraints over restating implementation.

### ✅ DO
```python
def parse_region(value: str) -> str:
    """Return normalized region code."""
```

### ✅ DO
```python
def acquire_lock(resource_id: int) -> None:
    """Create a lock record for the resource.

    Raises:
        IntegrityError: If the lock already exists.
    """
    Lock.objects.create(resource_id=resource_id)
```

### ❌ DO NOT
```python
def parse_region(value: str) -> str:
    """This function parses region."""
```
