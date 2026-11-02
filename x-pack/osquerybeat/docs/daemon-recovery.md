# Osquery daemon recovery

Unexpected process exits after the daemon has fetched validated configuration
are recoverable, including exit 78, other nonzero exits, and unexpected clean
exits. A process that exits before readiness remains terminal, so invalid
startup configurations are not retried indefinitely. Existing broken-pipe,
connection-reset, EOF, extension watchdog and network timeout failures use the
same bounded recovery policy.

At most five restarts may happen within any five minutes. Each restart waits
1 second, doubled for every earlier restart still within the last five minutes,
so a crash loop waits 1, 2, 4, 8, and 16 seconds. Restarts older than five
minutes no longer count, so failures spread further apart than that never
exhaust the budget, and the delay falls back as they age out. Policy changes do
not count as restarts. A crash loop exhausts the budget only if six failures
fall within five minutes, so a loop whose runs each last more than about a
minute keeps restarting and stays DEGRADED. Once the budget is exhausted, an
OTel receiver reports FAILED until the receiver or Agent is restarted. On the
process runtime the beat exits with code 1; Agent restarts it about 1 s later
with a fresh budget and reports the exit as STOPPED rather than FAILED, so a
persistent failure cycles through about 31 s of retries and a restart. These
limits are fixed defaults, not new configuration options.

Backoff is cancellable. Policy updates replace the pending configuration without
shortening the delay, and removing all inputs cancels pending restarts. Adding
inputs again starts osqueryd without waiting for the rest of the delay, but
keeps the restart count. No replacement process starts before the previous run
finishes cleanup.

Compared with earlier releases, exits after readiness with code 0, or with a
nonzero code other than 78, are now retried; they used to report FAILED at
once. Removing all inputs now stops osqueryd and unregisters the action
handler.

While recovering, osquerybeat reports DEGRADED to the process-runtime status.
Only readiness from the current process restores RUNNING; stale callbacks from
an earlier process cannot mark its replacement healthy. Startup failures and
exhausted retries report FAILED. An OTel receiver does not forward these beat
status updates to the collector host, so it shows FAILED only when the beat
stops with an error, such as after a startup failure or exhausted retries.

## Scope

This recovery policy does not change publishing, logger callbacks, profiling,
or shared queue admission. Stopping the extension server waits, with no
timeout, for logger callbacks that are still publishing. If an output outage
blocks a publication, the run does not finish until the output accepts the
event. Until then no replacement daemon starts, even if osqueryd has already
exited, and the process-runtime status keeps reporting RUNNING. Shutdown waits for the
same publication. This is the scenario reported in #53645, where the output is
down and osqueryd exits 78, so this change alone does not make that case report
DEGRADED and restart; #53939 is needed for that. Scheduled query profiling
already runs outside the logger callback, so a stuck profile publish does not
delay restart. Separating scheduled result publication from the logger
callback remains independent work.

## Validation

Unit regressions cover backoff, exhaustion, cancellation, the restart window,
latest-policy selection, startup and runtime exit classification, and status
reporting. Native receiver coverage exercises an actual exit-78
restart and requires matching osqueryd and Elastic extension binaries. A full
deployed Agent/Elasticsearch reproduction of the original macOS environment is
a separate validation step.
