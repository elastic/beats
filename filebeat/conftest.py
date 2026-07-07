import os
import sys

import pytest

sys.path.append(os.path.join(os.path.dirname(__file__), '../libbeat/tests/system'))


# Names of the module fileset test files. These are the only system tests that
# are safe to spread across pytest-xdist workers: each worker uses its own
# Elasticsearch index (see test_modules.py). Every other system test keeps its
# current serial semantics (fixed ports, shared fixtures, etc.).
_PARALLEL_SAFE_TESTS = ("test_modules.py", "test_xpack_modules.py")


def pytest_collection_modifyitems(config, items):
    """
    Pin every non-module system test to a single xdist worker.

    When the suite runs in parallel (``-n`` with ``--dist loadgroup``), tests
    sharing an ``xdist_group`` are guaranteed to run on the same worker.
    Assigning all non-module tests to one shared group keeps them serial (as
    today) while leaving the module fileset tests free to distribute across
    workers. Without pytest-xdist this hook is a no-op.
    """
    if not config.pluginmanager.hasplugin("xdist"):
        return
    if not config.getoption("numprocesses", None):
        return
    for item in items:
        if any(name in item.nodeid for name in _PARALLEL_SAFE_TESTS):
            continue
        item.add_marker(pytest.mark.xdist_group("serial"))
