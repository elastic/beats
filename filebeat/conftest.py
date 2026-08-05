import os
import re
import sys

import pytest

sys.path.append(os.path.join(os.path.dirname(__file__), '../libbeat/tests/system'))


# Names of the module fileset test files. These are the only system tests that
# are safe to spread across pytest-xdist workers: each worker uses its own
# Elasticsearch index (see test_modules.py). Every other system test keeps its
# current serial semantics (fixed ports, shared fixtures, etc.).
_PARALLEL_SAFE_TESTS = ("test_modules.py", "test_xpack_modules.py")

# parameterized.expand names each fileset case after its position and its first
# parameter, the module: ``test_fileset_file_025_postgresql``. The fileset is
# not part of the name, so the module is the finest key available here.
_FILESET_CASE = re.compile(r"test_fileset_file_\d+_([a-z0-9]+)")


def _fileset_group(nodeid):
    """Return the xdist group for a module fileset test, None if unrecognized."""
    match = _FILESET_CASE.search(nodeid)
    if match is None:
        return None
    return "module-" + match.group(1)


@pytest.hookimpl(tryfirst=True)
def pytest_collection_modifyitems(config, items):
    """
    Assign xdist groups so parallel workers cannot corrupt each other's state.

    When the suite runs in parallel (``-n`` with ``--dist loadgroup``), tests
    sharing an ``xdist_group`` are guaranteed to run on the same worker.
    Without pytest-xdist this hook is a no-op.

    Non-module tests all share one group, keeping their existing serial
    semantics (fixed ports, shared fixtures, etc.).

    Module fileset tests get one group per module. A worker's Elasticsearch
    index is its own, but a module's ingest pipelines are cluster-global and
    every test case starts a Filebeat that loads them. That is safe when the
    loads succeed, since they are idempotent PUTs, but ``LoadPipelines``
    (filebeat/fileset/pipelines.go) rolls back on a failed load by *deleting*
    the pipelines it loaded. A transient failure in one worker therefore
    deletes pipelines a concurrent worker is actively ingesting through,
    which surfaces as either "Pipeline processor configured for non-existent
    pipeline [...]" on the indexed documents or, when the deleted pipeline is
    the fileset's entry point, as events that never get indexed at all and a
    test that times out waiting for its index to appear. Confining a module to
    one worker serializes those loads and removes the race. Different modules
    still run in parallel, which is where the bulk of the speedup comes from.

    ``tryfirst`` is required: pytest-xdist's worker turns the ``xdist_group``
    marker into the ``@group`` nodeid suffix it schedules on from its own
    ``pytest_collection_modifyitems``, so ours must add the marker before that
    runs. The grouping happens during collection on the workers, where the
    ``numprocesses``/``dist`` options are not set (they live on the controller),
    so detect an active run via ``PYTEST_XDIST_WORKER`` too.
    """
    distributing = bool(os.environ.get("PYTEST_XDIST_WORKER")) or bool(
        config.getoption("numprocesses", None))
    if not distributing:
        return
    for item in items:
        if any(name in item.nodeid for name in _PARALLEL_SAFE_TESTS):
            # Fall back to the serial group for anything in a module test file
            # that is not a recognized fileset case: sharing pipelines across
            # workers is the hazard, so an unknown case must not run free.
            item.add_marker(pytest.mark.xdist_group(
                _fileset_group(item.nodeid) or "serial"))
            continue
        item.add_marker(pytest.mark.xdist_group("serial"))
