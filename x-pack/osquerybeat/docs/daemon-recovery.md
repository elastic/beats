# Osquery daemon recovery

Osquerybeat restarts osqueryd after exit 78 only when that process has fetched
its validated input configuration. Exit 78 before readiness remains terminal,
including when a replacement process fails during startup.

Recovery uses the existing immediate restart path, as do recognized broken-pipe,
connection-reset and EOF errors. This change
adds no backoff or retry limit and leaves existing health reporting unchanged.
Repeated runtime failures can therefore cause repeated immediate restarts.

Restart waits for the previous run to finish cleanup. A logger callback blocked
on output publication can still delay recovery; callback and profiling fixes
are separate changes.
