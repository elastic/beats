# Osquery daemon recovery

Osquerybeat restarts osqueryd after exit 78 only when that process has fetched
its validated input configuration. Exit 78 before readiness remains terminal,
including when a replacement process fails during startup.

Recovery uses the existing immediate restart path, as do recognized broken-pipe,
connection-reset, EOF, extension watchdog and network timeout errors. This change
adds no backoff or retry limit and leaves existing health reporting unchanged.
Repeated runtime failures can therefore cause repeated immediate restarts.

Restart waits for the previous run to finish cleanup. Output publishing no
longer runs in the logger callback; see [publishing.md](publishing.md) for what
can still wait for the output.
