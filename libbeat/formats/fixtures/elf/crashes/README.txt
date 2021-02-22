This directory contains a sample of binaries that currently
result in code that panics, these were obtained via fuzzing.
They happen as a result of oversized allocations in `debug/elf`
that don't guard against invalid segment lengths passed in to a
`make(..., segment.Size)` call. For the code not to panic, this
should be fixed upstream.
