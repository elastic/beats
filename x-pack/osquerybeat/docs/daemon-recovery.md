# Osquery output backpressure and daemon recovery

Native scheduled-query logger callbacks do not wait for type resolution, output
queue capacity, or profiling. They admit work to a bounded FIFO owned by the
osquerybeat instance. A single worker handles results across daemon restarts,
including zero-hit completion responses. Completion responses precede optional
profiling. Live-query and RRULE publishing retain their existing delivery modes.

The handoff holds at most 128 outstanding result documents and an estimated
8 MiB of decoded result storage, including the document being processed. Raw
JSON larger than 8 MiB is dropped before decoding. These bounds limit the result
backlog, not total process memory or transient decoding allocations. Scheduled
publishing is best effort: full queues, oversized documents, retired policies,
and cancelled runs can discard data. Warnings are rate limited; monitoring
counters under `osquerybeat.scheduled_results.dropped` report `overflow`, `stale`,
`shutdown`, and `output_full`. The first three count result documents, while
`output_full` counts events rejected by the output queue.

Each policy owns its scheduled publishing clients. Replacement invalidates old
work, including when a namespace is reused. Deferred query identity and routing
must match the current policy before work is admitted. Closing or replacing a
policy does not wait for an active publication; retirement prevents subsequent
rows from using its clients. One in-flight `beat.Client.Publish` can still wait
for output recovery or shared pipeline closure. Shutdown does not join that
worker before the pipeline has had a chance to release it. No additional
publication workers accumulate across daemon restarts.

## Recovery policy

Unexpected process exits after the daemon has fetched validated configuration
are recoverable, including exit 78, other nonzero exits, and unexpected clean
exits. A process that exits before readiness remains terminal, so invalid
startup configurations are not retried indefinitely. Existing broken-pipe,
connection-reset, EOF, extension watchdog and network timeout failures use the
same bounded recovery policy.

The initial run may be followed by five replacement attempts. The delays are
1, 2, 4, 8, and 16 seconds; exponential backoff has a 30-second upper bound.
The restart budget and delay reset only after a process has remained ready for
five minutes. Merely fetching configuration, or repeatedly changing policy,
does not reset the budget. Persistent failures exhaust it and report FAILED.
An operator must then correct the cause and restart the failed receiver or
Agent. These limits are fixed defaults, not new configuration options.

Backoff is cancellable. Policy updates replace the pending configuration without
shortening the delay, and removing all inputs cancels pending restarts. No
replacement process starts before the previous run finishes cleanup. While
recovering, process-runtime and per-input OTel status reporters receive DEGRADED.
Only readiness from the current process restores RUNNING; stale callbacks from
an earlier process cannot mark its replacement healthy. Startup failures and
exhausted retries report FAILED to both status paths.

## Validation

Unit regressions cover real memory-queue saturation, logger admission, shutdown,
byte and document budgets, retirement between rows, cancelled daemon runs,
policy replacement, backoff, exhaustion, stable-period reset, latest-policy
selection and per-input OTel status forwarding. Scoped suites run with the race
detector. Native receiver regressions cover output outage/recovery with default
profiling and an actual exit-78 restart; they require the matching osqueryd and
Elastic extension binaries. A full deployed Agent/Elasticsearch reproduction of
the original macOS environment is a separate validation step.
