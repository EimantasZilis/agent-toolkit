# Go concurrency

## Apply this first

- Start a goroutine only when its lifetime, cancellation, ownership, and
  completion signal are explicit. Avoid fire-and-forget work in request paths.
- Prefer communicating ownership through channels when that makes the design
  clearer; use mutexes for small, directly shared state and document the
  protected invariants.
- Pass cancellation and deadlines through `context.Context`; check them in
  loops and before blocking or expensive work.
- Close channels only by the sender that owns the send side. Do not close a
  channel merely to signal receivers when cancellation or a separate done
  channel is clearer.
- Test concurrent code with `go test -race`; design for deterministic shutdown
  rather than relying on scheduler timing.

~~~go
func worker(ctx context.Context, jobs <-chan Job, results chan<- Result) {
    for {
        select {
        case <-ctx.Done(): return
        case job, ok := <-jobs:
            if !ok { return }
            results <- process(ctx, job)
        }
    }
}
~~~

## Quick checklist

- [ ] Every goroutine has a bounded lifetime and an owner.
- [ ] Shared state has one clear synchronization strategy.
- [ ] Cancellation, backpressure, and channel ownership are explicit.
- [ ] Shutdown cannot leak goroutines or deadlock on blocked sends.
- [ ] Relevant tests pass under the race detector.

## DO / DO NOT

~~~go
// DO: protect a compound invariant with one mutex.
mu.Lock()
defer mu.Unlock()
if cache.ready { return cache.value }

// DO NOT: start work without a way to stop or observe it.
go refreshCacheForever()
~~~

## Allowed exceptions

Long-lived goroutines are valid for process-wide supervisors, servers, and
metrics collectors when their owner has a documented shutdown path. A lock is
preferable to a channel when the state is small and the critical section is
obvious; record that choice when it is not self-evident.

## Sources

- [Effective Go: concurrency](https://go.dev/doc/effective_go#concurrency) (accessed 2026-09-30)
- [Go Code Review Comments: contexts](https://go.dev/wiki/CodeReviewComments#contexts) (accessed 2026-09-30)
- [Race detector](https://go.dev/doc/articles/race_detector) (accessed 2026-09-30)
