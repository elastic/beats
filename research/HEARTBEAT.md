# Heartbeat Receiver: DirectQueue Analysis

## Status: SAFE for DirectQueue

Heartbeat can safely use DirectQueue. The architecture prevents both
scheduler stalls and goroutine accumulation.

## Architecture: one scheduler, independent monitor goroutines

Heartbeat has **one central scheduler** for all monitors (unlike metricbeat
which has one ticker per metricset). All monitors register their jobs on the
same scheduler via `sched.Add`. The scheduler manages a timer queue and
dispatches tasks when their scheduled time arrives.

The critical design: each task runs in its **own goroutine**, and the **next
run is only scheduled after the current run completes**.

## Event flow

```
Scheduler timer thread (single goroutine, never blocks on tasks)
  │
  │  timer fires for monitor "http-check-1"
  │
  └─ go taskFn(now)                              ← NEW GOROUTINE
      │
      └─ sj.run()                                ← BLOCKS until task completes
          │
          ├─ scheduler.limitSem.Acquire()         ← global concurrency limit
          ├─ jobLimitSem.Acquire()                ← per-type limit (if set)
          │
          └─ sj.runTask(entrypoint)
              │
              └─ task(ctx)                        ← executes monitor check
                  └─ runPublishJob(job, pubClient)
                      ├─ job(event)               ← HTTP/TCP/ICMP check runs
                      └─ pubClient.Publish(event) ← WITH DIRECTQUEUE: BLOCKS
                  └─ return continuations (if any)
              │
              └─ for each continuation:
                  go sj.runTask(cont)             ← parallel goroutines
          │
          └─ sj.wg.Wait()                        ← wait for all continuations
          └─ limitSem.Release()
      │
      └─ s.runTaskOnce(sched.Next(lastRanAt), taskFn, true)
      │   ↑ NEXT RUN SCHEDULED ONLY HERE — AFTER sj.run() RETURNS
      │
      └─ goroutine exits
```

## Why blocking Publish is safe

### 1. The scheduler thread is never blocked

`scheduler.go:250`:
```go
asyncTask := func(now time.Time) { go taskFn(now) }
s.timerQueue.Push(runAt, asyncTask)
```

The timer thread pushes tasks and moves on. It never waits for a task
goroutine. If monitor A's Publish blocks, the timer thread continues
dispatching monitor B, C, D on schedule.

### 2. No goroutine accumulation — at most one goroutine per monitor

`scheduler.go:204-215`:
```go
lastRanAt = sj.run()           // BLOCKS until task + continuations complete
// ...
s.runTaskOnce(sched.Next(lastRanAt), taskFn, true)  // schedule NEXT run
```

The next run is only scheduled **after** `sj.run()` returns. And `sj.run()`
blocks at `sj.wg.Wait()` until the task and all continuations finish. So if
Publish blocks:

- `sj.run()` never returns
- `runTaskOnce` for the next interval is never called
- **No new goroutine is spawned for that monitor**
- The monitor simply skips intervals until the current Publish completes

This means there is at most **one active goroutine per monitor** at any time.
There is no risk of goroutine accumulation from a slow OTel pipeline.

### 3. Monitors are independent

Each monitor has:
- Its own `configuredJob` with its own task closure
- Its own goroutine (spawned per execution)
- Its own pipeline client (`pubClient`)
- Its own schedule (registered independently on the timer queue)

If monitor A's Publish blocks, monitor B's timer fires independently and
runs in its own goroutine. The only shared resource is the concurrency
semaphores.

### 4. Backpressure via semaphores (not goroutine pileup)

Two semaphores gate concurrent execution:

- **Global limit** (`scheduler.limitSem`) — shared by all monitors.
  Default: unlimited.
- **Per-type limit** (`jobLimitSem`) — per monitor type.
  Default: unlimited for http/tcp/icmp, 2 for browser.

If a monitor holds a semaphore slot while blocked on Publish, it prevents
another monitor (of the same type, or globally) from starting. This is
correct backpressure — the same thing that happens when the queue is full
in DefaultQueue mode.

Since each monitor has at most one active goroutine, the maximum number of
held semaphore slots equals the number of monitors currently executing. A
slow OTel pipeline causes monitors to hold slots longer, which delays other
monitors. When the pipeline recovers, slots are released and monitors
resume normal scheduling.

### 5. What happens under sustained OTel pipeline slowness

```
10 HTTP monitors, 10s period each, OTel pipeline takes 8s per event:

t=0s:   all 10 monitors fire, 10 goroutines spawn
t=0s:   all 10 start HTTP checks (fast)
t=0.1s: all 10 call Publish — all block for ~8s
t=8.1s: all 10 Publishes complete
t=8.1s: all 10 schedule next run at t=10s
t=10s:  all 10 fire again
...

No accumulation. Each monitor runs once, waits, schedules next.
If Publish takes longer than the period, the monitor simply runs
less frequently — it never double-fires.
```

## run_once mode

In `run_once` mode, heartbeat wraps the pipeline with `SyncPipelineWrapper`
which tracks events via a WaitGroup (`wg.Add(1)` on Publish, `wg.Add(-1)`
on ACK). With DirectQueue, ACK happens synchronously inside Publish (via
`directBatch.ACK`), so the WaitGroup increments and decrements within the
same call. `pipelineWrapper.Wait()` returns correctly after all events are
published.

## Comparison

| Aspect | DefaultQueue | DirectQueue |
|--------|-------------|-------------|
| Scheduler blocks | No | No (goroutine per task) |
| Monitor goroutine blocks | No (queue buffers) | Yes (until OTel accepts) |
| Goroutine accumulation | No | No (next run only after current completes) |
| Other monitors affected | No | Only via semaphore contention |
| Missed intervals | No (queue absorbs) | Yes (monitor skips if Publish slow) |
| run_once mode | Works | Works |
| Backpressure | Queue full | Semaphore + blocked goroutine |

## Code locations

| Component | File | Lines |
|-----------|------|-------|
| Single scheduler creation | `heartbeat/beater/heartbeat.go` | 119 |
| All monitors share `sched.Add` | `heartbeat/beater/heartbeat.go` | 134 |
| Task goroutine spawn | `heartbeat/scheduler/scheduler.go` | 250 |
| Next run scheduled after completion | `heartbeat/scheduler/scheduler.go` | 204-215 |
| Task execution + wg.Wait | `heartbeat/scheduler/schedjob.go` | 59-76 |
| Global semaphore acquire | `heartbeat/scheduler/schedjob.go` | 95 |
| Per-type semaphore acquire | `heartbeat/scheduler/schedjob.go` | 63 |
| Event publish | `heartbeat/monitors/task.go` | 125, 128 |
| SyncPipelineWrapper | `heartbeat/monitors/pipeline.go` | 43-92 |
| Concurrency limits config | `heartbeat/config/config.go` | 59-88 |
