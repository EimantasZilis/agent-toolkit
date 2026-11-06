# Imports

## Apply this first

- Group imports: stdlib, third-party, then local.
- Avoid wildcard imports.
- Prefer one import per line for readability.
- Use relative imports within the same package.
- Expose intended package API via `__init__.py`.
- Avoid forcing consumers to import deep internal modules.

## Quick checklist

- [ ] Imports are grouped stdlib -> third-party -> local.
- [ ] No wildcard imports.
- [ ] Same-package imports use the established local convention.
- [ ] Any local function-level import has a documented reason.
- [ ] Public package API is exposed intentionally via `__init__.py`.

---

- stdlib -> third-party -> local
- local imports preferred
- no wildcard imports

## Generic rules
### ✅ DO
```python
import datetime
from django.db import transaction
from billing.models import Invoice
...
```

### ❌ DO NOT
```python
from billing.models import *
import django, datetime
...
```

---

## Imports within the same packages
### ✅ DO use relative imports within the same packages
```python
from .models import Invoice
from .services.payment import process_payment
...
```

### ❌ DO NOT use absolute imports for local modules within the same package
```python
from billing.models import Invoice
from billing.services.payment import process_payment
...
```

---

## Expose public API via `__init__.py`
### ✅ DO
```python
# some_package/__init__.py
# import datetime
from .hello import world
```

# so that consumers import from package root via
```python
from some_package import world
...
```

### ❌ DO NOT force consumers to import from deep internal paths
```python
from some_package.hello import world
```

---

## Additional import hygiene

- Keep imports at module top-level (after module docstring), except for deliberate cycle-breaking or optional dependency gates.
- If a local import is required inside a function, add a short comment stating why.
- Prefer explicit imports over alias-heavy patterns unless the alias is conventional (`import numpy as np` or `import pandas as pd`).
- Keep import side effects out of module import time where possible.

### ✅ DO
```python
def serialize_payload(payload: dict[str, object]) -> str:
    # Local import avoids optional dependency at module import time.
    import orjson

    return orjson.dumps(payload).decode("utf-8")
```

### ❌ DO NOT
```python
def serialize_payload(payload):
    import orjson
    return orjson.dumps(payload)
```
(missing type hints and no reason for local import)

---
