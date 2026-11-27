# Result publishing

Osquerybeat publishes native scheduled query results from an internal queue
with its own worker and its own output clients. The worker uses blocking
`beat.Client.Publish`, so under output back-pressure it waits for queue slots
alongside other receivers, as publishing did before; what changes is where it
waits. osqueryd's logger callback only adds results to the scheduled queue and
never waits for the output, so a slow output can no longer hold the logger RPC
until osqueryd times out.

## Settings

The settings are read from the result input's
`osquery.elastic_options.publishing`:

```yaml
publishing:
  scheduled:
    max_events: 3200   # rows; unset or 0 uses the default
    max_age: 0         # unset or 0 disables it
```

Negative values are rejected, and the policy that contains them is not
applied. Durations are Go durations such as `60s`, or bare integers in
seconds; a quoted number without a unit, such as `"60"`, is rejected.
`max_events` has no upper bound, like `queue.mem.events`, and the memory the
queue can use grows with it.

## Scheduled queue

The scheduled queue holds native scheduled results. An item is one query run:
its rows plus its response document.

`max_events` limits the rows in the queue, like the memory queue's
`queue.mem.events`. Response documents count as one row each. When an item
does not fit, the oldest queued items are dropped until it does. An item larger
than `max_events` is still accepted whole once everything older has been
dropped, so a large result is not cut while the output is healthy. The item the
worker is publishing does not count against the limit and is never dropped for
space. A policy change applies the new limit; shrinking it drops the oldest
items.

In a burst, rows are dropped only once libbeat's queue is full and more than
`max_events` rows (3200 by default) wait here; the queue does not wait before
it drops them, because a wait would hold osqueryd's logger RPC again.

A dropped differential result is lost for good. osqueryd updates its diff
baseline before it logs the result, so it does not report those added or
removed rows again. A snapshot query, the default, reports all its rows on
its next run.

`max_age` drops items that waited longer than this when the worker reaches
them.

A policy change keeps the queued items. Each item carries the namespace and
query captured when it was queued. When the worker reaches it, it publishes it
through the clients of the current policy if that policy still has the query
with the same namespace and SQL, and drops it otherwise. A policy change retires
the previous clients. The item in flight keeps publishing on them while the new
policy still has its query with the same namespace and SQL; otherwise its rows
not yet handed to the output are dropped. One `Publish` call can still be
blocked on the retired clients until the output recovers or the pipeline
closes.

The queue also survives osqueryd restarts, such as an options or extensions
change or a crash. The worker resolves a native result's column types when it
publishes the result, with the client of the osqueryd run that logged it or,
once that run has ended, of the current run. While no osqueryd runs, the worker
keeps that result at the head of the queue and waits; new items queue behind
it, and it can still be dropped for space, `max_age` or shutdown. When an
osqueryd run ends, queued items the current policy no longer publishes are
dropped. The rows and response of a native result whose column types osqueryd
could not resolve are dropped.

Shutdown drops the queued items and does not wait for a blocked `Publish`.

## Metrics and logs

The queue reports under `osquerybeat.publishing.scheduled`:

| Metric | Meaning |
|---|---|
| `dropped.<reason>` | Dropped items |
| `dropped_rows.<reason>` | Rows of the dropped items |
| `queue.items` | Items waiting in the queue |
| `queue.events` | Rows waiting in the queue |

Reasons:

| Reason | Meaning |
|---|---|
| `overflow` | Dropped for space, oldest first |
| `expired` | Older than `max_age` when the worker reached it |
| `stale` | No current policy publishes it: the query was removed, moved to another namespace or changed SQL (also while it was in flight), osqueryd could not resolve its column types, or no results stream is configured |
| `shutdown` | Osquerybeat was stopping |

An event the pipeline discards while closing counts in
`dropped_rows.shutdown` only. The queue logs at most one warning per minute
about drops, naming the counter that moved.

## Behaviour changes

- An osquery policy that fails validation, for example one with invalid
  publishing settings, an invalid processor, an invalid ECS mapping or an
  invalid query schedule, is now rejected before any of it is applied. On the process runtime the
  osquery unit is marked failed, the previous policy keeps running, and the
  next valid policy clears the failure. Before, a configuration that failed
  to parse was ignored and the unit stayed healthy, and a policy osquerybeat
  could not apply, such as one with an invalid ECS mapping, stopped
  osquerybeat. On the OTel runtime the policy is the receiver's static
  configuration and is not checked this way: a configuration that fails to
  parse stops the receiver from starting, and one osquerybeat cannot apply
  makes the receiver report a failed status and stop, as before.
- Queued scheduled results survive osqueryd restarts (an options or
  extensions change, or a crash) and are published once osqueryd runs again.
- Results still queued when osquerybeat stops are dropped and counted as
  `shutdown`.
