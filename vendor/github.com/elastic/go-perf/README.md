----

This is a clone of the `golang.org/x/sys/unix/linux/perf` submitted by
[acln0](https://github.com/acln0) to review at
https://go-review.googlesource.com/c/sys/+/168059

An alternative working tree for this package can also be found
at https://github.com/acln0/perf

This Elastic fork contains bugfixes and features necessary for
our KProbes implementation.

----

`perf` API client package for Linux. See `man 2 perf_event_open` and
`include/uapi/linux/perf_event.h`.

This package is in its early stages. The API is still under discussion:
it may change at any moment, without prior notice. Furthermore,
this document may not be completely up to date at all times.


Testing
=======

Many of the things package perf does require elevated privileges on
most systems. We would very much like for the tests to not require
root to run. Because of this, we use a fairly specific testing model,
described next.

If the host kernel does not support `perf_event_open(2)` (i.e. if
the `/proc/sys/kernel/perf_event_paranoid` file is not present),
then tests fail immediately with an error message.

Tests are designed in such a way that they are skipped if their
requirements are not met by the underlying system. We would like the
test suite to degrade gracefully, under certain circumstances.

For example, when running Linux in a virtualized environment, various
hardware PMUs might not be available. In such situations, we would like
the test suite to continue running. For this purpose, we introduce the
mechanism described next.

Requirements for a test are specified by invoking the `requires`
function, at the beginning of a test function. All tests that call
`perf_event_open` must specify requirements this way. Currently,
we use three kinds of requirements:

* `perf_event_paranoid` values

* the existence of various PMUs (e.g. "cpu", "software", "tracepoint")

* tracefs is mounted, and readable

Today, setting `perf_event_paranoid=1` and having a readable tracefs
mounted at `/sys/kernel/debug/tracing` enables most of the tests.
A select few require `perf_event_paranoid=0`. If the test process
is running with `CAP_SYS_ADMIN`, `perf_event_paranoid` requirements
are ignored, since they are considered fulfilled. The test process
does not attempt to see if it is running as root, it only checks
`CAP_SYS_ADMIN`.

If you find a test that, when ran without elevated privileges,
fails with something akin to a permissions error, then it means the
requirements for the test were not specified precisely. Please file
a bug. Extending the test suite and making these requirements more
precise is an ongoing process.
