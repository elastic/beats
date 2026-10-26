# Native scheduled-result publishing

Native scheduled-query logger callbacks admit results to a bounded FIFO owned
by the osquerybeat instance. One worker resolves result types and publishes rows
and completion responses across daemon restarts. A slow output does not hold the
logger RPC. RRULE-scheduled runs use the same queue and worker, with their rows
already resolved. Live-query publishing retains its existing delivery mode.

The handoff holds at most 128 outstanding result documents and an estimated
8 MiB of decoded result storage, including the active document. Raw JSON larger
than 8 MiB is dropped before decoding. These bounds limit retained result data,
not total process memory or transient decoding allocations. Scheduled publishing
is best effort: full queues, oversized documents, results older than
`publishing.scheduled.max_age`, moved queries, and cancelled runs can discard
data. The worker's clients block for space in the output queue rather than
dropping events. Warnings are rate limited. Monitoring counters under
`osquerybeat.scheduled_results.dropped` report `overflow`, `stale`, `expired`,
and `shutdown`, and `osquerybeat.scheduled_results.dropped_rows` reports the
rows of the `overflow`, `stale`, and `expired` drops.

Each policy owns separate result, completion and profile publishing clients. Replacement
keeps queued work; the worker publishes each result through the policy active
when it is dequeued, if that policy still routes the query to the same namespace
and SQL, and drops it as `stale` otherwise. Namespace and query metadata are
captured together and checked against the current policy before admission.
Replacement prevents subsequent rows from using the old clients. One active
`beat.Client.Publish` can still wait
for output recovery or shared pipeline closure. Shutdown stops admission and
retires clients without joining that call. Daemon restarts reuse the same worker
instead of accumulating blocked publication goroutines.

Profiling keeps its separate bounded worker, which collects profiles and queues
each one for the scheduled worker as a one-row item. Admission captures one
response ID and execution slot for both. A blocked result publication does not
hold profile collection, and slow profiling does not hold the result worker.
On overflow, queued profiles are evicted before results. Profile drops are also
counted under `osquerybeat.scheduled_query_profiles.dropped`: `output_full` for
overflow, `stale`, and `shutdown`. An RRULE run's profile is queued with the run
and carries `schedule_id` and `schedule_execution_count`. Profile events can
arrive before result events; correlation uses the captured identity, not
delivery order. Native profile measurements remain a latest snapshot, not exact
historical measurements for the originating execution.

This change does not alter daemon restart or backoff policy. Unit regressions
cover real memory-queue saturation, result bounds, shutdown, policy retirement,
run cancellation, correlation, and independence from profiling. Native receiver
tests cover output outage recovery without changing the daemon PID and shutdown
while output remains unavailable, with default profiling enabled. Matching
osqueryd and Elastic extension binaries are required for those tests.
