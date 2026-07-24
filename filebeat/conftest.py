import os
import sys

import pytest

sys.path.append(os.path.join(os.path.dirname(__file__), '../libbeat/tests/system'))


# Names of the module fileset test files. These are the only system tests that
# are safe to spread across pytest-xdist workers: each worker uses its own
# Elasticsearch index (see test_modules.py). Every other system test keeps its
# current serial semantics (fixed ports, shared fixtures, etc.).
_PARALLEL_SAFE_TESTS = ("test_modules.py", "test_xpack_modules.py")


@pytest.hookimpl(tryfirst=True)
def pytest_collection_modifyitems(config, items):
    """
    Pin every non-module system test to a single xdist worker.

    When the suite runs in parallel (``-n`` with ``--dist loadgroup``), tests
    sharing an ``xdist_group`` are guaranteed to run on the same worker.
    Assigning all non-module tests to one shared group keeps them serial (as
    today) while leaving the module fileset tests free to distribute across
    workers. Without pytest-xdist this hook is a no-op.

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
            continue
        item.add_marker(pytest.mark.xdist_group("serial"))
