# Osquerybeat Receiver: DirectQueue Analysis

## Status: NOT SAFE for DirectQueue

Osquerybeat cannot safely use DirectQueue without code changes. The osquery
daemon's extension callback mechanism creates a blocking chain that would
stall the daemon if `client.Publish` is synchronous.

## Event flow: osquery daemon to output

### The blocking chain

```
osqueryd subprocess (has extension call timeout)
  └─ executes scheduled query
  └─ calls logger plugin via thrift over unix socket (BLOCKING — waits for return)
      └─ osquery-go extension server goroutine
          └─ loggerPlugin.Log(ctx, LogTypeSnapshot, jsonBytes)
              └─ handleQueryResult(ctx, cli, configPlugin, res)
                  └─ configPlugin.LookupQueryInfo(res.Name)       ← RWMutex read, fast
                  └─ cli.ResolveResult(ctx, query, res.Hits)      ← may query osqueryd
                  └─ bt.pub.Publish(index, ..., hits, ...)
                      └─ p.mx.Lock()                              ← acquires Publisher mutex
                      └─ for each hit:
                          p.client.Publish(event)                 ← WITH DIRECTQUEUE: BLOCKS
                      └─ p.mx.Unlock()
                  └─ bt.pub.PublishQueryProfile(...)              ← also acquires mutex
                  └─ bt.pub.PublishScheduledResponse(...)         ← also acquires mutex
              ← handleQueryResult returns
          ← loggerPlugin.Log returns
      ← thrift response sent to osqueryd
  └─ osqueryd resumes
```

Every step runs synchronously on the same goroutine. If `client.Publish`
blocks (DirectQueue), the entire chain stalls until the OTel pipeline
accepts the event.

### Why this is dangerous

1. **osqueryd has an extension call timeout** (`extensions_timeout` flag).
   If the logger plugin callback exceeds this timeout, osqueryd may kill the
   extension or restart. Blocking on a slow OTel pipeline could trigger this.

2. **The Publisher mutex is held during all blocking publishes.** Three
   different goroutines compete for this single `sync.Mutex`:

   | Source | Goroutine | Methods |
   |--------|-----------|---------|
   | Scheduled query results | Extension server goroutine | `Publish`, `PublishQueryProfile`, `PublishScheduledResponse` |
   | Action results | Agent manager goroutine | `PublishActionResult` |
   | Config changes | Main beat loop | `Configure` (recreates clients) |

   With DirectQueue, a blocking `Publish` holds the mutex and blocks all
   other publishers including config updates.

3. **Potential circular dependency.** `handleQueryResult` calls
   `cli.ResolveResult()` which queries osqueryd BEFORE publishing. If the
   publisher goroutine and extension goroutine share the same context,
   osqueryd could be blocked waiting for a callback while the publisher is
   blocked waiting for osqueryd.

## What would need to change

Decouple the extension callback from publishing by queuing results on a
channel:

```
Current:
  extension goroutine → handleQueryResult → Publisher.Publish (blocks)

Needed:
  extension goroutine → resultQueue <- result       (returns immediately)
  publisher goroutine → result := <-resultQueue     (dedicated goroutine)
                      → handleQueryResult            (blocks on Publish — safe)
```

### Specific changes (~30 lines in osquerybeat.go)

1. Add a `chan QueryResult` to the `osquerybeat` struct
2. Logger plugin callback sends to channel instead of calling
   `handleQueryResult` directly — callback returns immediately
3. A dedicated goroutine reads from the channel and calls
   `handleQueryResult` → `Publisher.Publish`
4. The dedicated goroutine is free to block on DirectQueue without
   affecting the osquery daemon

### Risk: ResolveResult dependency

`handleQueryResult` calls `cli.ResolveResult(ctx, query, hits)` which may
query osqueryd. If this runs in the publisher goroutine (after decoupling),
verify that it doesn't create a circular dependency:

```
publisher goroutine:
  handleQueryResult()
    → cli.ResolveResult()        ← queries osqueryd
                                    osqueryd is processing next query
                                    osqueryd calls logger plugin callback
                                    callback tries to send to resultQueue
                                    resultQueue is unbuffered/full?
                                    → DEADLOCK
```

Mitigation: use a buffered channel for the result queue, or move
`ResolveResult` before the queue send (keep it in the extension goroutine).

## Code locations

| Component | File | Lines |
|-----------|------|-------|
| Main run loop | `beater/osquerybeat.go` | 191-338 |
| handleQueryResult | `beater/osquerybeat.go` | 562-632 |
| Logger plugin callback | `beater/logger_plugin.go` | 46-71 |
| Extension server setup | `beater/osquerybeat.go` | 497-522 |
| osqueryd subprocess | `internal/osqd/osqueryd.go` | 203-282 |
| Publisher mutex + Publish | `internal/pub/publisher.go` | 29, 156-165 |
| PublishActionResult | `internal/pub/publisher.go` | 185-199 |
| Configure (client swap) | `internal/pub/publisher.go` | 49-154 |
| Action handler | `beater/action_handler.go` | 77-94, 114-174 |
