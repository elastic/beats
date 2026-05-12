# Metricbeat & Auditbeat Receiver: DirectQueue + Batched Runner

## Overview

When running as beatreceivers in an OTel Collector, metricbeat and auditbeat
use two optimizations that work together:

1. **DirectQueue** — bypasses the publisher queue, forwarding events synchronously
   to the OTel consumer inline in the Publish call.
2. **Batched Runner** — synchronizes all periodic metricsets in a module on a
   single ticker, fetches them in parallel, then sends each metricset's events
   via `client.PublishAll` instead of one at a time.

Combined, this gives one `ConsumeLogs` call per metricset per fetch cycle
instead of one per event, matching the OTel receiver pattern where scrapers
batch all data into a single `Consume*` call.

Auditbeat uses the metricbeat beater framework (`beater.CreatorWithRegistry`),
so the entire flow described here applies identically to both.

## Event flow: standalone vs receiver mode

### Standalone mode (DefaultQueue, per-metricset runners)

```
Per module (e.g. "system" with cpu, memory, disk):

  Each metricset gets its own Runner + Wrapper + Client:

  cpu runner:
    cpu goroutine ──→ reporter.Event(e) ──→ writeEvent ──→ channel (cap=1)
    PublishChannels goroutine ──→ client.Publish(e) ──→ queue ──→ consumer ──→ worker ──→ otelConsumer
                                                         one ConsumeLogs per event

  memory runner:
    memory goroutine ──→ ... same independent chain ...

  disk runner:
    disk goroutine ──→ ... same independent chain ...

  Each metricset has its own ticker. Fetch cycles are not synchronized.
  Events interleave across metricsets. Each event produces one ConsumeLogs call.
```

### Receiver mode (DirectQueue, batched runner)

```
Per module (e.g. "system" with cpu, memory, disk):

  All periodic metricsets share one batchedRunner with a single ticker:

  batchedRunner ticker fires
    ├── go cpu.fetch(bufferingReporter)     ← parallel goroutines
    ├── go memory.fetch(bufferingReporter)
    └── go disk.fetch(bufferingReporter)
    ... wait for all to complete ...
    ├── cpuClient.PublishAll(cpuEvents)      ← one ConsumeLogs per metricset
    ├── memoryClient.PublishAll(memEvents)
    └── diskClient.PublishAll(diskEvents)
```

## How it works in detail

### Step 1: Factory creates a batched runner

When `WithBatchedMode()` is set on the beater, `Factory.Create` partitions
metricsets into push vs periodic:

- **Periodic metricsets** (`ReportingMetricSetV2*`) → collected into one
  `batchedRunner` with a single ticker for the module period.
- **Push metricsets** (`PushMetricSetV2*`) → get individual standard runners
  (same as standalone mode) since they run indefinitely and cannot be
  synchronized.

Each metricset still gets its own `beat.Client` with its own processors
(including per-metricset processors from light module manifests). This
preserves processor correctness.

### Step 2: Buffering reporter collects events during fetch

Instead of the channel-based `eventReporter` that writes each event to a
`chan beat.Event`, the batched runner uses a `bufferingReporter` that appends
events to a `[]beat.Event` slice:

```go
// Channel-based (standalone):
func (r reporterV2) Event(event mb.Event) bool {
    beatEvent := r.msw.toPublishEvent(event, r.start)
    writeEvent(r.done, r.out, beatEvent)  // blocks on channel
}

// Buffering (receiver):
func (r *bufferingReporterV2) Event(event mb.Event) bool {
    beatEvent := r.msw.toPublishEvent(event, r.start)
    r.buf = append(r.buf, beatEvent)       // no blocking
}
```

Both use the same `toPublishEvent` helper for event enrichment (timing,
period, host, namespace, stats, event modifiers).

### Step 3: Parallel fetch, then batch publish

On each tick, the batched runner:

1. Creates a `bufferingReporter` per metricset
2. Launches all metricset fetches in parallel (one goroutine each, with panic
   recovery)
3. Waits for all fetches to complete
4. For each metricset, calls `client.PublishAll(events)` with that metricset's
   buffered events

```go
func (br *batchedRunner) fetchAndPublish(ctx *channelContext) {
    // Create reporters
    reporters := make([]*bufferingReporter, len(br.msws))
    for i := range br.msws {
        reporters[i] = &bufferingReporter{msw: br.msws[i], done: br.done}
    }

    // Fetch all in parallel
    var wg sync.WaitGroup
    for i, msw := range br.msws {
        wg.Add(1)
        go func(i int, msw *metricSetWrapper) {
            defer wg.Done()
            reporters[i].StartFetchTimer()
            msw.fetch(ctx, reporters[i])
        }(i, msw)
    }
    wg.Wait()

    // Publish each metricset's events as a batch
    for i, r := range reporters {
        if events := r.flush(); len(events) > 0 {
            br.clients[i].PublishAll(events)
        }
    }
}
```

### Step 4: PublishAll through DirectQueue

`client.PublishAll` runs processors on each event, then checks if the
producer implements `BatchProducer`. The `directProducer` does, so all
processed events are sent as a single `directBatch` → one
`otelConsumer.Publish` call → one `ConsumeLogs` call containing all events
as log records.

```
client.PublishAll(events)
  ├── processEvent(e1) → run processors
  ├── processEvent(e2) → run processors
  ├── processEvent(e3) → run processors
  └── producer.PublishAll([e1, e2, e3])
        └── flush([e1, e2, e3])
              └── otelConsumer.Publish(ctx, directBatch{[e1, e2, e3]})
                    └── ConsumeLogs(pLogs)   ← one call, 3 log records
```

## Comparison

| Aspect | Standalone | Receiver |
|--------|-----------|----------|
| Ticker | One per metricset (independent) | One per module (synchronized) |
| Fetch | Independent goroutines, staggered start | Parallel goroutines, start together |
| Event buffering | Channel (cap=1) per module | Slice per metricset per cycle |
| Publish call | `client.Publish` per event | `client.PublishAll` per metricset |
| ConsumeLogs calls | One per event | One per metricset per cycle |
| Processors | Per-metricset client (correct) | Per-metricset client (correct) |
| Push metricsets | Standard runner | Standard runner (unchanged) |
| Queue | memqueue (3200 events) | None (DirectQueue) |
| Workers | NumCPU otelConsumer workers | None (inline) |
| Backpressure | Queue full blocks producers | PublishAll blocks fetch cycle |

## Example: system module with cpu + memory + diskio

### Standalone (10s period, 4 disks)

```
t=0.0s  cpu fetch    → 1 event  → Publish → ConsumeLogs  (1 record)
t=0.2s  memory fetch → 1 event  → Publish → ConsumeLogs  (1 record)
t=0.5s  diskio fetch → 5 events → Publish × 5 → ConsumeLogs × 5  (1 record each)
t=10.0s cpu fetch ...
t=10.2s memory fetch ...
...
Total: 7 ConsumeLogs calls per cycle
```

### Receiver (10s period, 4 disks)

```
t=0.0s  all fetch in parallel:
          cpu    → buffers 1 event
          memory → buffers 1 event
          diskio → buffers 5 events
        all done:
          cpuClient.PublishAll([1 event])      → ConsumeLogs (1 record)
          memoryClient.PublishAll([1 event])    → ConsumeLogs (1 record)
          diskioClient.PublishAll([5 events])   → ConsumeLogs (5 records)
t=10.0s next cycle...
...
Total: 3 ConsumeLogs calls per cycle
```

## Flag propagation

```
mbreceiver/factory.go
  └── beater.Creator(..., beater.WithBatchedMode())
        └── Metricbeat.batchedMode = true
              └── module.NewBatchedFactory(...)
                    └── Factory.batchedMode = true
                          └── Factory.Create() → createBatched()
                                └── batchedRunner for periodic metricsets
                                    + standard runners for push metricsets
```
