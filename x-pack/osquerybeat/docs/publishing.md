# Result publishing

Osquerybeat publishes osquery results from two internal queues, each with its
own worker and its own output clients: one for scheduled queries and one for
live queries. The workers use blocking `beat.Client.Publish`, so under output
back-pressure they wait for queue slots alongside other receivers instead of
dropping events. osqueryd's logger callback only adds results to the scheduled
queue and never waits for the output, so a slow output cannot hold the logger
RPC until osqueryd times out.

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
applied.

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
the previous clients; rows of an item in flight that were not yet handed to the
output when this happens are dropped. One `Publish` call can still be blocked
on the retired clients until the output recovers or the pipeline closes.
Results of an osqueryd run that has ended are dropped too.

Shutdown drops the queued items and does not wait for a blocked `Publish`.

## Live queue

A live query action waits for its rows to be handed to the output for at most
`live.timeout`, counted from when the query returns, so a slow query does not
use up the window. If the timeout elapses, the action reports the rows handed
over so far as its count, with an error. The remaining rows are skipped; a row
already inside a blocked `Publish` can still be delivered late. The action's
response document is published after its rows and is not dropped by the
timeout, so Kibana always learns the outcome. If osquerybeat is stopping, the
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
| `stale` | both | No current policy publishes it: the query was removed, moved to another namespace or changed SQL, the policy was replaced while it was in flight, its osqueryd run ended, or no results stream is configured |
| `shutdown` | both | Osquerybeat was stopping |
| `timeout` | live | The action stopped waiting after `live.timeout` |

For a live action, an item is the action's results, and its rows are the rows
not handed to the output; the live queue gauges show the waiting action. An
event the pipeline discards while closing counts in `dropped_rows.shutdown`
only. Each queue logs at most one warning per minute about drops, naming the
counter that moved.

Scheduled profile drops are also counted under the earlier
`osquerybeat.scheduled_query_profiles.dropped` counters, which existing
dashboards read: `output_full` now counts profiles dropped for space in the
scheduled queue (it used to count profiles a full output refused), and
`stale`, `expired` and `shutdown` count the other drops. `queue_full` still
counts profiles the collection worker had no room for.

## Behaviour changes

- Any invalid osquery input configuration, including invalid publishing
  settings, now marks the osquery unit as failed. Before, an invalid
  configuration was ignored and the unit stayed healthy.
- Live query actions that fail now report their real `started_at` and
  `completed_at`.
- The 128-document and 8 MiB bounds on scheduled results are gone; the queue is
  bounded by `max_events` rows only. Memory per row is the same as with the
  memory queue's `queue.mem.events`.
