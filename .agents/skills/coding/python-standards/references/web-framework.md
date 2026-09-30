# Django and DRF guidance

## Apply this first

- Keep a view narrow: validate input, call a service, and return the response.
- Put domain behavior in services rather than views or serializers.
- Use `transaction.atomic` when several writes must succeed together.
- Prefer the decorator when the whole function is transactional; use an inline
  block for a deliberately smaller scope.
- Use bulk operations when they preserve correctness.
- Preload related data with `select_related` or `prefetch_related` to avoid
  N+1 queries.
- Limit serializers to validation and representation.

## Quick checklist

- [ ] Views follow validate -> service -> response.
- [ ] Business rules do not live in views or serializers.
- [ ] Multi-write operations have an appropriate transaction boundary.
- [ ] Related querysets avoid N+1 access.
- [ ] Serializers handle validation and representation only.
- [ ] Bulk operations preserve the required behavior and signals.

## View boundaries

Views should coordinate the request lifecycle, not implement the domain flow.

### ✅ DO

```python
def submit_invoice_view(request: Request) -> Response:
    form = InvoiceSerializer(data=request.data)
    form.is_valid(raise_exception=True)

    invoice = submit_invoice(**form.validated_data)

    return Response(InvoiceSerializer(invoice).data)
```

### ❌ DO NOT

```python
def submit_invoice_view(request):
    invoice = Invoice.objects.create(...)
```

The view now owns persistence and bypasses the service boundary.

## Services and atomic writes

Wrap a multi-write service in `transaction.atomic` so partial success cannot
leave inconsistent state. Bulk creation is useful when it preserves the
service's correctness requirements.

### ✅ DO

```python
from collections.abc import Iterable

from django.db import transaction


@transaction.atomic
def open_batch(owner: dict[str, str], labels: Iterable[str]) -> Batch:
    batch = Batch.objects.create(owner=owner)
    BatchEntry.objects.bulk_create(
        [BatchEntry(batch=batch, label=label) for label in labels]
    )

    return batch
```

### Allowed exception

Use an inline `with transaction.atomic():` when only part of a larger function
needs a transaction, or when nesting/lifetime makes the narrower boundary
clearer. Do not force the decorator form in that case.

## Query loading

Preload related objects when the code will access them repeatedly.

### ✅ DO

```python
Invoice.objects.select_related("owner").all()
```

### ❌ DO NOT

```python
Invoice.objects.all()
```

The latter can trigger one query per invoice when each owner is later read.

## Serializer boundaries

Serializers validate and represent data; they should not contain business
workflow or domain decisions.

### ❌ DO NOT

```python
class InvoiceSerializer(serializers.Serializer):
    def create(self, validated_data):
        # Domain workflow belongs in a service.
        ...
```

### Allowed exception

Serializer-level field validation and representation logic are appropriate.
Move behavior that coordinates domain actions, persistence, or multiple models
to a service.
