## Logging rules

## Apply this first

- Log events that help diagnose failures and behavior.
- Include stable context fields (`extra`) for correlation.
- Never log secrets, tokens, passwords, or credentials.
- Prefer structured context over interpolated sensitive strings.
- Use `logger.exception(...)` when handling exceptions.

## Quick checklist

- [ ] Message names are stable and event-oriented.
- [ ] Variable context is structured in `extra`.
- [ ] Correlation IDs are included when available.
- [ ] No sensitive values are logged.
- [ ] Log level matches event severity and volume.

---

- Log useful context
- Never log secrets or credentials

### ✅ DO
```python
logger.exception("Failed order", extra={"order_id": order.id})
```

### ❌ DO NOT
```python
logger.error(f"password={user.password}")
```

---

## Structured logging conventions

- Prefer stable event-style messages and place variable context in structured fields (`extra`), not in free-form message strings.
- Include correlation identifiers when available (`request_id`, `task_id`, `user_id`, etc.).
- Choose levels consistently:
  - `debug` for diagnostic detail
  - `info` for important lifecycle events
  - `warning` for degraded-but-recoverable paths
  - `error`/`exception` for failures
- Avoid high-volume logs inside tight loops unless guarded or sampled.

### ✅ DO
```python
logger.info(
    "task_processed",
    extra={"task_id": task.id, "duration_ms": duration_ms},
)
```

### ❌ DO NOT
```python
logger.info(f"Processed task {task.id} in {duration_ms} ms")
```