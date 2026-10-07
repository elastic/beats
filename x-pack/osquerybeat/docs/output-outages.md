# Output outages and scheduled result delivery

Osquerybeat accepts native scheduled results through a worker owned by each
osqueryd run. The logger callback does not resolve column types, publish events,
or query `osquery_schedule`. Accepted results are processed in order. The
handoff allows up to 128 outstanding result documents, including the one being
processed, and up to 8 MiB of estimated decoded storage. Estimates include row,
map, field, and string storage. Result JSON larger than 8 MiB is rejected before
parsing. These limits bound retained work; they are not a process memory limit.

When the handoff is full, osquerybeat drops the new result and acknowledges the
logger call so output backpressure cannot cause osqueryd to shut down. Results,
action responses (including scheduled runs with zero hits), and profiles use
nonblocking publisher queue admission. Events are dropped if the queue cannot
accept them. This is best-effort delivery: prolonged output outages can lose
results, responses, and profiles. Already admitted events retain the publisher's
normal output behavior. Increasing the queue can absorb longer bursts but does
not guarantee retention through an indefinite outage.

The following monitoring counters report losses and queue admission:

- `osquerybeat.scheduled_results.dropped.overflow`: full handoff or oversized result.
- `osquerybeat.scheduled_results.dropped.stale`: unavailable query metadata or an
  obsolete policy generation.
- `osquerybeat.scheduled_results.dropped.shutdown`: pending work discarded during
  shutdown, including callbacks received after cancellation.
- `osquerybeat.publishing.<dataset>.accepted`: events admitted to the publisher
  queue. Admission does not mean Elasticsearch has acknowledged delivery.
- `osquerybeat.publishing.<dataset>.dropped`: events rejected at queue admission.
  Datasets are `osquery_manager.result`, `osquery_manager.action.responses`, and
  `osquery_manager.query_profile`.

Warnings are rate limited to once per minute for handoff losses and once per
minute per publishing dataset. Scheduled results capture query, mapping,
namespace, and schedule metadata when accepted. A publisher policy change
invalidates pending work so old results cannot use replacement clients or
permissions. osqueryd metadata retains its existing configuration refresh timing.

Scheduled response documents are submitted before profile collection. Profiles
sample the latest `osquery_schedule` row after the logger callback has returned.
A later query execution may update that row before sampling; `response_id` links
the triggering result and is not a guarantee of an exact historical measurement.
Profile queries share the result worker, whose count and byte bounds remain in
force while profiling waits. Cancelling a run interrupts client waits and RPC
transport reads, and queued work is discarded before the worker is joined.

If osqueryd exits with status 78, or the extension transport fails with a
recoverable disconnect or timeout, osquerybeat reports `Degraded` and retries
with exponential backoff from 1 second to 30 seconds. Configuration updates
received during backoff apply to the next run. The status returns to `Running`
when that run applies its generated configuration. Backoff resets after a ready
run remains stable for one minute. Invalid configuration, failed installation,
and unclassified process exits remain terminal errors.
