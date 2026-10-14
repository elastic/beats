# Osquery daemon recovery

Osquerybeat restarts osqueryd after exit 78 when that daemon has fetched its
validated input configuration. Exit 78 before that point remains a startup
failure, since retrying invalid startup configuration would conceal the error.
Existing recoverable transport failures use the same retry policy.

Recovery allows five retries, delayed by 1, 2, 4, 8, and 16 seconds. Repeated
failures exhaust this budget and fail the component. The budget resets only
after a daemon has remained ready for one minute; reaching readiness briefly
does not reset a crash loop. These limits are internal and introduce no new
configuration settings.

The component reports degraded health when a ready daemon exits, including
while its previous run finishes shutting down. It returns to running when the
replacement daemon fetches its validated configuration. These states
reach both Elastic Agent process management and the OTel receiver's active
input reporters. Inputs added during recovery inherit its status; removed
inputs stop receiving updates.

With no inputs, the Agent considers the beat running, but this idle status does
not mark osqueryd ready. The first input or an input re-added after removal waits
for the current daemon to fetch configuration before reporting healthy.

Configuration updates received during backoff are retained for the next run.
Removing every input suspends recovery. Shutdown cancels the backoff without
starting another daemon.

Restart waits for the previous run to finish shutting down. A logger callback
blocked on output publication can still delay that cleanup. This change does
not prevent output outages from blocking logger callbacks or resolve the
separate scheduled-profiling issue. It does not change result delivery, result
size limits, or the shared queue memory budget.
