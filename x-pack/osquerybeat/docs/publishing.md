# Result publishing

Osquerybeat publishes osquery results from two internal queues, each with its
own worker and its own output clients: one for scheduled queries and one for
live queries. The workers use blocking `beat.Client.Publish`, so under output
back-pressure they wait for queue slots alongside other receivers, as
publishing did before; what changes is where they wait. osqueryd's logger
callback only adds results to the scheduled queue and never waits for the
output, so a slow output can no longer hold the logger RPC until osqueryd
times out.

## Settings

The settings are read from the result input's
`osquery.elastic_options.publishing`:

```yaml
publishing:
  scheduled:
    max_events: 3200   # rows; unset or 0 uses the default
    max_age: 0         # unset or 0 disables it
  live:
    timeout: 60s       # unset or 0 uses the default
```

Negative values are rejected, and the policy that contains them is not
applied. Durations are Go durations such as `60s`, or bare integers in
seconds; a quoted number without a unit, such as `"60"`, is rejected.
`max_events` has no upper bound, like `queue.mem.events`, and the memory the
queue can use grows with it.

## Scheduled queue

The scheduled queue holds native scheduled results, RRULE-scheduled runs and
scheduled query profiles. An item is one query run: its rows plus its response
document. An RRULE run's profile is part of its item; a native run's profile is
collected by a separate bounded worker and queued as its own one-row item.

`max_events` limits the rows in the queue, like the memory queue's
`queue.mem.events`. Response and profile documents count as one row each. When
an item does not fit, the oldest queued items are dropped until it does:
queued profiles first, then results. An item larger than `max_events` is still
accepted whole once everything older has been dropped, so a large result is not
cut while the output is healthy. The item the worker is publishing does not
count against the limit and is never dropped for space. A policy change applies
the new limit; shrinking it drops the oldest items.

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

## Live queue

A live query action waits for its rows to be handed to the output for at most
`live.timeout`, counted from when the query returns, so a slow query does not
use up the window. If the timeout elapses, the action reports the rows handed
over so far as its count, with an error. The remaining rows are skipped; a row
already inside a blocked `Publish` can still be delivered late. The action's
response document is published after its rows and is not dropped by the
timeout, so Kibana learns the outcome unless osquerybeat stops before the
response is published. If osquerybeat is stopping, the
action reports an error instead of a count. If no results stream is configured,
the action reports that error instead.

Live actions run one at a time. While one waits, `Attach` and `Clear` (called
when osqueryd is re-registered) wait too, each queued action for up to its
query time plus `live.timeout`. With several actions queued, that adds up to
roughly N times (query time + `live.timeout`), rather than a single timeout.

A live query that fails but asked for a profile publishes the profile through
the live queue and can wait up to `live.timeout` for it before reporting the
query error.

## Metrics and logs

Each queue reports under `osquerybeat.publishing.scheduled` and
`osquerybeat.publishing.live`:

| Metric | Meaning |
|---|---|
| `dropped.<reason>` | Dropped items |
| `dropped_rows.<reason>` | Rows of the dropped items |
| `queue.items` | Items waiting in the queue |
| `queue.events` | Rows waiting in the queue |

Reasons:

| Reason | Queues | Meaning |
|---|---|---|
| `overflow` | scheduled | Dropped for space, oldest first and profiles first |
| `expired` | scheduled | Older than `max_age` when the worker reached it |
| `stale` | both | No current policy publishes it: the query was removed, moved to another namespace or changed SQL (also while it was in flight), osqueryd could not resolve its column types, no results stream is configured, or, for a profile or an action response, no profile or action response stream is configured |
| `shutdown` | both | Osquerybeat was stopping |
| `timeout` | live | The action stopped waiting after `live.timeout` |

For a live action, an item is the action's results and profile, and its rows
are the rows and profile not handed to the output. The live `queue.*` gauges
cover only the waiting action, not the action responses queued behind it. A
live action response dropped because osquerybeat is stopping, or because no
live clients or no action response stream are configured, counts as its own
item of one row. An event the pipeline discards while closing counts in
`dropped_rows.shutdown` only. Each queue logs at most one warning per minute
about drops, naming the counter that moved.

A scheduled profile dropped once it is queued is counted only under
`osquerybeat.publishing.scheduled`. The profile collection worker reports
under `osquerybeat.scheduled_query_profiles`: `dropped.queue_full`,
`dropped.stale` and `dropped.shutdown` count profiles dropped before they were
queued, `failed` counts failed collections, and `collected` counts profiles
collected and queued, not delivered.

## Behaviour changes

- Any invalid osquery input configuration, including invalid publishing
  settings, is now rejected. On the process runtime the osquery unit is
  marked failed and the previous settings stay active; before, an invalid
  configuration was ignored and the unit stayed healthy. On the OTel runtime
  the receiver fails to start.
- Live query actions that fail now report their real `started_at` and
  `completed_at`.
- Live query action responses are published asynchronously by the live
  worker, after the action's rows, instead of synchronously when the action
  returns.
- Live query rows not handed to the output within `live.timeout` are skipped
  and the action reports a partial count with an error. Before, they were
  delivered eventually, once the blocked publish went through.
- Queued scheduled results survive osqueryd restarts (an options or
  extensions change, or a crash) and are published once osqueryd runs again.
- Results still queued when osquerybeat stops are dropped and counted as
  `shutdown`.
