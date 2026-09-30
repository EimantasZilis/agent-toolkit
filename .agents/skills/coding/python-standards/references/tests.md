# Python testing guidance: defaults and justified exceptions

## Apply this first

- Use pytest tests that describe observable behavior.
- Keep setup, action, and assertion phases visibly separate.
- Prefer fixtures or factories, and parameterize value-only variations.
- Add GIVEN/WHEN/THEN documentation to non-trivial behavior tests.
- Mock external boundaries only; state database baselines when persistence is
  part of the behavior.

## Quick checklist

- [ ] The test name states the behavior under test.
- [ ] Setup, action, and assertions are easy to distinguish.
- [ ] Non-trivial tests include a useful GIVEN/WHEN/THEN docstring.
- [ ] Blank lines separate meaningful phases.
- [ ] Fixtures/factories are used where appropriate.
- [ ] Parameterization covers cases that differ only by values.
- [ ] Mocks target external boundaries, not internal logic.
- [ ] Persistence tests state their initial count when it matters.
- [ ] Flakiness risks from time, paths, ordering, and shared state are avoided.
- [ ] Test functions return `None` explicitly or by annotation.

## Working agreement

- Use pytest.
- Name tests for the behavior they establish.
- Keep the phases of a test visible: setup, action, then assertions.
- Favor fixtures and factories over verbose hand-built records.
- Parameterize cases that vary only in their inputs or expected outputs.
- Patch boundaries outside the unit under test; leave its internal decisions real.
- State the initial database count when persistence is the behavior being checked.
- These are defaults, not absolute prohibitions. Deviate only for a concrete
  reason, such as genuinely different setup, a small pure value object, or a
  local repository convention.

## Database effects

When saving records is central to the behavior, make the baseline explicit.

### ✅ DO

```python
def test_register_job_adds_one_record() -> None:
    assert Job.objects.count() == 0

    register_job(label="daily")

    assert Job.objects.count() == 1
```

### Acceptable exception

```python
def test_encoder_keeps_priority_field() -> None:
    job = JobFactory()
    document = encode_job(job)

    assert "priority" in document
```

No count baseline is needed here because persistence is incidental to the
serialization assertion.

### ❌ DO NOT

```python
def test_register_job() -> None:
    register_job(label="daily")
    assert Job.objects.count() == 1
```

This relies on an unstated initial state even though the initial count matters.

## Names and intent

Test names should explain the observable result. Type test functions and use
`-> None` explicitly or by annotation when they do not return a value.

### ✅ DO

```python
def test_rendering_pending_job_marks_it_inactive() -> None:
    job = JobFactory(state="pending")

    payload = render_job(job)

    assert payload["active"] is False
```

### ❌ DO NOT

```python
def test_case_2() -> None:
    assert render_job(Job(state="pending"))["active"] == False
```

The latter hides the behavior in a generic name and compresses setup, action,
and assertion into one expression.

## Fixtures and factories

Use the repository's fixture or factory support when it removes irrelevant
construction noise.

### ✅ DO

```python
from factory import Factory


def test_job_belongs_to_batch(batch_factory: Factory) -> None:
    batch = batch_factory()
    job = JobFactory(batch=batch)

    assert job.batch_id == batch.id
```

### Acceptable exception

```python
def test_retry_policy_is_a_small_value_object() -> None:
    policy = RetryPolicy(limit=4, pause_seconds=2)

    assert policy.limit == 4
```

Manual construction is reasonable for a tiny pure object with no database
behavior.

### ❌ DO NOT

```python
def test_job() -> None:
    batch = Batch.objects.create(title="b")
    job = Job.objects.create(batch=batch)

    assert job.batch.title == "b"
```

This bypasses available factories without a stated reason.

## Setup, action, assertion

Leave a blank line between meaningful phases. The following makes each phase
scannable:

```python
def test_endpoint_reports_pending_jobs(http_client: Client) -> None:
    JobFactory(state="pending")

    result = http_client.get("/jobs/")

    assert result.status_code == 200
    assert result.json()["total"] == 1
```

Avoid hiding setup or the action inside an assertion:

```python
def test_endpoint() -> None:
    assert http_client.get("/jobs/").json()["total"] == 1
```

## GIVEN / WHEN / THEN documentation

Add a short GIVEN/WHEN/THEN docstring to tests that contain branching,
database or filesystem effects, external boundaries, several meaningful setup
steps, or an integration-style flow. A one-line assertion over a tiny pure
value object is the main exception; it does not excuse omitting intent from a
test that otherwise needs explanation.

### ✅ DO

```python
def test_creating_a_job_persists_one_record() -> None:
    """
    GIVEN the job store has no records
    WHEN a job is created
    THEN the store contains exactly one record
    """
    assert Job.objects.count() == 0

    JobFactory()

    assert Job.objects.count() == 1
```

The docstring names the behavior; blank lines in the body still show setup,
action, and assertion.

### ❌ DO NOT use vague documentation

```python
def test_job_encoder() -> None:
    """
    GIVEN database setup
    WHEN encoder runs
    THEN query returns JSON
    """
```

That description does not identify the actual behavior under test.

## Parameterized cases

Use one parameterized test when only the values change.

### ✅ DO

```python
@pytest.mark.parametrize(
    "raw_value,expected_score",
    [
        ("ready", 1),
        ("", 0),
        (None, 0),
    ],
)
def test_score_for_status(raw_value: object, expected_score: int) -> None:
    assert score_for_status(raw_value) == expected_score
```

### ❌ DO NOT duplicate literal-only cases

```python
def test_missing_status() -> None:
    assert score_for_status(None) == 0


def test_blank_status() -> None:
    assert score_for_status("") == 0
```

Separate tests are fine when the behavior or setup differs materially:

```python
def test_score_retries_after_network_timeout() -> None:
    ...


def test_score_does_not_retry_invalid_status() -> None:
    ...
```

## Boundary mocking

Mock HTTP, I/O, time, environment access, or UUID generation when those are
external boundaries. Do not replace the unit's internal logic with mocks.
Patch where the dependency is used.

### ✅ DO

```python
from unittest.mock import Mock, patch


@patch("portal.handlers.fetch_remote_status")
def test_handler_returns_remote_status(mock_fetch: Mock) -> None:
    mock_fetch.return_value = {"state": "ready"}

    response = client.get("/status/")

    assert response.status_code == 200
```

### Acceptable environment boundary

```python
def test_region_comes_from_environment(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("SERVICE_REGION", "ap-southeast-2")

    assert read_region() == "ap-southeast-2"
```

## Completion checklist

- [ ] The name states the tested behavior.
- [ ] Setup, action, and assertion are easy to distinguish.
- [ ] A non-trivial behavior test has a useful GIVEN/WHEN/THEN docstring.
- [ ] Blank lines divide meaningful test phases.
- [ ] Appropriate fixtures or factories are used.
- [ ] Mocks are limited to external boundaries.
- [ ] Time, shared paths, ordering, and other flakiness sources are controlled.
- [ ] A persistence test states its count baseline.

### ❌ DO NOT patch internal helpers

```python
@patch("console.services._format_locally")
def test_console_service(mock_formatter: Mock) -> None:
    ...
```

This substitutes internal implementation details rather than an external
boundary.

## Naming patched dependencies

Give patched objects descriptive names. A module constant can make a patch
target easier to scan when several tests share it.

```python
SERVICE_MODULE = "console.service"


@patch(f"{SERVICE_MODULE}.send_notification")
def test_notification_is_sent(mock_send_notification: Mock) -> None:
    pass
```

Avoid opaque names such as `m`:

```python
@patch("console.service.send_notification")
def test_notification_is_sent(m: Mock) -> None:
    pass
```

## Grouping class methods

Arrange methods by behavior or topic. Alphabetical order is optional, but an
unexplained mixture of lifecycle actions makes a class harder to scan.

```python
class TestJobLifecycle:
    def test_create_job(self): ...
    def test_start_job(self): ...
    def test_fail_job(self): ...
```

This is also acceptable when the methods form a coherent topic:

```python
class TestJobLifecycle:
    def test_create_job(self): ...
    def test_remove_job(self): ...
    def test_update_job(self): ...
```

## Coverage focus

Prefer a test that covers a meaningful transition when a behavior has several
states:

```python
def test_job_moves_through_processing_states() -> None:
    job = JobFactory(state="queued")

    job.start()
    assert job.state == "running"

    job.fail()
    assert job.state == "failed"
```

A focused test is still appropriate when it covers a distinct behavior:

```python
def test_starting_a_job_sets_running_state() -> None:
    job = JobFactory(state="queued")
    job.start()

    assert job.state == "running"
```

## Contract versus implementation

Assert the public result or contract, not the object's private representation.

```python
def test_job_document_exposes_contract_keys() -> None:
    job = JobFactory()

    document = render_job(job)

    assert {"id", "state", "active"} <= set(document)
```

Avoid comparing the full result with an internal attribute dictionary:

```python
def test_job_renderer() -> None:
    job = JobFactory()

    assert render_job(job) == job.__dict__
```

That couples the test to representation details rather than the external
contract.

## Pytest integration hygiene

- Follow pytest discovery names (`test_*.py` and `test_*` functions/classes).
- Keep tests deterministic: avoid order, wall-clock, network, and shared
  mutable-global dependencies.
- Use `tmp_path` or `tmp_path_factory` instead of real project paths for files.
- Use `monkeypatch` for process-level environment/config changes so cleanup is
  automatic.
- Mark or separate slow and integration tests according to repository rules so
  the default suite stays fast.

```python
def test_report_is_written_to_isolated_directory(tmp_path: Path) -> None:
    destination = tmp_path / "summary.json"
    write_report(destination, {"complete": True})

    assert destination.exists()
```

Avoid shared paths such as `/tmp/report.json`, which can make tests flaky:

```python
def test_report_is_written() -> None:
    write_report(Path("/tmp/report.json"), {"complete": True})
```
