# Journald tests (Debian 12)
The tests for the journald input (currently only used for Debian 12
testing) require journal files (test files ending in `.journal`), those
files are generated using `systemd-journal-remote` (see the [Journald
input README.md](../../input/journald/README.md) for more details).

The source for those journal files are the `.export` files in the test
folder. Those files are the raw output of `journalctl -o export`. They
are added here because journal files format change with different
versions of journald, which can cause `journalclt` to fail reading
them, which leads to test failures. So if tests start failing because
`journalctl` cannot read the journal files as expected, new ones can
easily be generated with the same version of journalctl used on CI
and the original dataset.
