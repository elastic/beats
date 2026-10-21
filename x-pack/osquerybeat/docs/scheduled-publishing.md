# Native scheduled-result publishing

Native scheduled-query logger callbacks admit results to a bounded FIFO owned
by the osquerybeat instance. One worker resolves result types and publishes rows
and completion responses across daemon restarts. A slow output does not hold the
logger RPC. Live-query and RRULE publishing retain their existing delivery modes.

The handoff holds at most 128 outstanding result documents and an estimated
8 MiB of decoded result storage, including the active document. Raw JSON larger
than 8 MiB is dropped before decoding. These bounds limit retained result data,
not total process memory or transient decoding allocations. Scheduled publishing
is best effort: full queues, oversized documents, retired policies, and cancelled
runs can discard data. Warnings are rate limited. Monitoring counters under
`osquerybeat.scheduled_results.dropped` report `overflow`, `stale`, `shutdown`,
and `output_full`. The first three count documents; `output_full` counts events
rejected by the output queue.

Each policy owns separate result and completion publishing clients. Replacement
retires queued work and prevents subsequent rows from using the old clients.
Namespace and query metadata are captured together and checked against the
current policy before admission. One active `beat.Client.Publish` can still wait
for output recovery or shared pipeline closure. Shutdown stops admission and
retires clients without joining that call. Daemon restarts reuse the same worker
instead of accumulating blocked publication goroutines.

Profiling keeps its separate bounded collection worker, isolated RPC client,
and publication worker. Admission captures one response ID and execution slot
for both paths. A blocked result publication does not hold profile collection,
and slow profiling does not hold the result worker. Profile events can arrive
before result events; correlation uses the captured identity, not delivery order.
Native profile measurements remain a latest snapshot, not exact historical
measurements for the originating execution. No queue capacity or priority is
reserved for profiling.

This change does not alter daemon restart or backoff policy. Unit regressions
cover real memory-queue saturation, result bounds, shutdown, policy retirement,
run cancellation, correlation, and independence from profiling. Native receiver
tests cover output outage recovery without changing the daemon PID and shutdown
while output remains unavailable, with default profiling enabled. Matching
osqueryd and Elastic extension binaries are required for those tests.
