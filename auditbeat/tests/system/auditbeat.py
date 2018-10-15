import os
import shutil
import sys
import tempfile

sys.path.append(os.path.join(os.path.dirname(__file__), '../../../metricbeat/tests/system'))

if os.name == "nt":
    import win32file

from metricbeat import BaseTest as MetricbeatTest


class BaseTest(MetricbeatTest):
    @classmethod
    def setUpClass(self):
        self.beat_name = "auditbeat"
        self.beat_path = os.path.abspath(
            os.path.join(os.path.dirname(__file__), "../../"))
        super(MetricbeatTest, self).setUpClass()

    def create_file(self, path, contents):
        f = open(path, 'wb')
        f.write(contents)
        f.close()

    def check_event(self, event, expected):
        for key in expected:
            assert key in event, "key '{0}' not found in event".format(key)
            assert event[key] == expected[key], \
                "key '{0}' has value '{1}', expected '{2}'".format(key,
                                                                   event[key],
                                                                   expected[key])

    def temp_dir(self, prefix):
        # os.path.realpath resolves any symlinks in path. Necessary for macOS
        # where /var is a symlink to /private/var
        p = os.path.realpath(tempfile.mkdtemp(prefix))
        if os.name == "nt":
            # Under windows, get rid of any ~1 in path (short path)
            p = str(win32file.GetLongPathName(p))
        return p


class PathCleanup:
    def __init__(self, paths):
        self.paths = paths

    def __enter__(self):
        pass

    def __exit__(self, exc_type, exc_val, exc_tb):
        for path in self.paths:
            shutil.rmtree(path)
