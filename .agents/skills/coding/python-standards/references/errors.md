# Error handling in Python

## Apply this first

- Keep the protected operation inside a small, deliberate `try` block.
- Catch only errors the code can handle intentionally; never use bare
  `except:`.
- Combine exception types only when their response is identical.
- Give remapped errors domain-appropriate meaning and retain their cause.
- Never turn an exception path into a silent `pass`.

## Quick checklist

- [ ] The `try` region contains only the operation that may fail.
- [ ] Caught exception classes are named explicitly.
- [ ] Different recovery behavior uses separate handlers.
- [ ] `raise ... from exc` preserves the original cause when translating.
- [ ] Broad catches are limited to safe process or task boundaries.
- [ ] Exceptions are not being used for ordinary branching.

## Catching and translating errors

Handle equivalent failures together and separate cases when their responses
differ. Add context when converting a low-level error into a domain error.

### ✅ DO

```python
try:
    count = int(raw_count)
except (TypeError, ValueError) as exc:
    raise InvalidCountError("Count must be an integer") from exc
```

### ❌ DO NOT

```python
try:
    ...
except:
    pass
```

The example hides both the failure type and the fact that the operation failed.

## Propagation and boundaries

Raise an exception that makes sense at the boundary where it is exposed, while
preserving the lower-level cause. Catch `Exception` broadly only at a process
or task boundary where the code can record context and fail safely. Normal
control flow should use conditions or return values instead of exceptions.

### ✅ DO

```python
try:
    document = json.loads(raw_document)
except json.JSONDecodeError as exc:
    raise MalformedDocumentError("Document is not valid JSON") from exc
```

### ❌ DO NOT

```python
try:
    document = json.loads(raw_document)
    execute_pipeline(document)
except Exception:
    return None
```

This catches unrelated failures, discards their cause, and silently converts
an error into a normal result.

### Allowed exception

A broad `Exception` handler is acceptable at a deliberate process/task boundary
when it logs or records the needed context and then fails safely. It is not a
justification for broad catches inside ordinary application logic.
