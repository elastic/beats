# Osquery daemon recovery

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

## Scope

This recovery policy does not change publishing, logger callbacks, profiling,
or shared queue admission. Stopping the extension server waits, with no
timeout, for logger callbacks that are still publishing. If an output outage
blocks a publication, the run does not finish until the output accepts the
event. Until then no replacement daemon starts, even if osqueryd has already
exited, and both status paths keep reporting RUNNING. Shutdown waits for the
same publication. This is the scenario reported in #53645, where the output is
down and osqueryd exits 78, so this change alone does not make that case report
DEGRADED and restart; #53939 is needed for that. Separating scheduled
publication and profiling from the logger callback remains independent work.

## Validation

Unit regressions cover backoff, exhaustion, cancellation, stable-period reset,
latest-policy selection, startup and runtime exit classification, and per-input
OTel status forwarding. Native receiver coverage exercises an actual exit-78
restart and requires matching osqueryd and Elastic extension binaries. A full
deployed Agent/Elasticsearch reproduction of the original macOS environment is
a separate validation step.
