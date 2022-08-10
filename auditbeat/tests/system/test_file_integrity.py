import time
import unittest
import platform
from auditbeat import *


# Escapes a path to match what's printed in the logs
def escape_path(path):
    return path.replace('\\', '\\\\')


def has_file(objs, path, sha1hash):
    found = False
    for obj in objs:
        if 'file.path' in obj and 'file.hash.sha1' in obj \
                and obj['file.path'].lower() == path.lower() and obj['file.hash.sha1'] == sha1hash:
            found = True
            break
    assert found, "File '{0}' with sha1sum '{1}' not found".format(path, sha1hash)


def has_dir(objs, path):
    found = False
    for obj in objs:
        if 'file.path' in obj and obj['file.path'].lower() == path.lower() and obj['file.type'] == "dir":
            found = True
            break
    assert found, "Dir '{0}' not found".format(path)


def file_events(objs, path, expected):
    evts = set()
    for obj in objs:
        if 'file.path' in obj and 'event.action' in obj and obj['file.path'].lower() == path.lower():
            if isinstance(obj['event.action'], list):
                evts = evts.union(set(obj['event.action']))
            else:
                evts.add(obj['event.action'])
    for wanted in set(expected):
        assert wanted in evts, "Event {0} for path '{1}' not found (got {2})".format(
            wanted, path, evts)


def wrap_except(expr):
    try:
        return expr()
    except IOError:
        return False


class Test(BaseTest):

    def wait_output(self, min_events):
        self.wait_until(lambda: wrap_except(lambda: len(self.read_output()) >= min_events))
        # wait for the number of lines in the file to stay constant for a second
        prev_lines = -1
        while True:
            num_lines = self.output_lines()
            if prev_lines < num_lines:
                prev_lines = num_lines
                time.sleep(1)
            else:
                break

    @unittest.skipIf(os.getenv("CI") is not None and platform.system() == 'Darwin',
                     'Flaky test: https://github.com/elastic/beats/issues/24678')
    def test_non_recursive(self):
        """
        file_integrity monitors watched directories (non recursive).
        """

        dirs = [self.temp_dir("auditbeat_test"),
                self.temp_dir("auditbeat_test")]

        with PathCleanup(dirs):
            self.render_config_template(
                modules=[{
                    "name": "file_integrity",
                    "extras": {
                        "paths": dirs,
                        "scan_at_start": False
                    }
                }],
            )
            proc = self.start_beat()

            # wait until the directories to watch are printed in the logs
            # this happens when the file_integrity module starts.
            # Case must be ignored under windows as capitalisation of paths
            # may differ
            self.wait_log_contains(escape_path(dirs[0]), max_timeout=30, ignore_case=True)

            file1 = os.path.join(dirs[0], 'file.txt')
            self.create_file(file1, "hello world!")

            file2 = os.path.join(dirs[1], 'file2.txt')
            self.create_file(file2, "Foo bar")

            # wait until file1 is reported before deleting. Otherwise the hash
            # might not be calculated
            self.wait_log_contains("\"path\":\"{0}\"".format(escape_path(file1)), ignore_case=True)

            os.unlink(file1)

            subdir = os.path.join(dirs[0], "subdir")
            os.mkdir(subdir)
            file3 = os.path.join(subdir, "other_file.txt")
            self.create_file(file3, "not reported.")

            # log entries are JSON formatted, this value shows up as an escaped json string.
            self.wait_log_contains("\\\"deleted\\\"")
            self.wait_log_contains("\"path\":\"{0}\"".format(escape_path(subdir)), ignore_case=True)
            self.wait_output(3)
            self.wait_until(lambda: any(
                'file.path' in obj and obj['file.path'].lower() == subdir.lower() for obj in self.read_output()))

            proc.check_kill_and_wait()
            self.assert_no_logged_warnings()

            # Ensure all Beater stages are used.
            assert self.log_contains("Setup Beat: auditbeat")
            assert self.log_contains("auditbeat start running")
            assert self.log_contains("auditbeat stopped")

            objs = self.read_output()

            has_file(objs, file1, "430ce34d020724ed75a196dfc2ad67c77772d169")
            has_file(objs, file2, "d23be250530a24be33069572db67995f21244c51")
            has_dir(objs, subdir)

            file_events(objs, file1, ['created', 'deleted'])
            file_events(objs, file2, ['created'])

            # assert file inside subdir is not reported
            assert self.log_contains(file3) is False

    @unittest.skipIf(os.getenv("BUILD_ID") is not None, "Skipped as flaky: https://github.com/elastic/beats/issues/7731")
    def test_recursive(self):
        """
        file_integrity monitors watched directories (recursive).
        """

        dirs = [self.temp_dir("auditbeat_test")]

        with PathCleanup(dirs):
            self.render_config_template(
                modules=[{
                    "name": "file_integrity",
                    "extras": {
                        "paths": dirs,
                        "scan_at_start": False,
                        "recursive": True
                    }
                }],
            )
            proc = self.start_beat()

            # wait until the directories to watch are printed in the logs
            # this happens when the file_integrity module starts
            self.wait_log_contains(escape_path(dirs[0]), max_timeout=30, ignore_case=True)
            self.wait_log_contains("\"recursive\":true")

            # auditbeat_test/subdir/
            subdir = os.path.join(dirs[0], "subdir")
            os.mkdir(subdir)
            # auditbeat_test/subdir/file.txt
            file1 = os.path.join(subdir, "file.txt")
            self.create_file(file1, "hello world!")

            # auditbeat_test/subdir/other/
            subdir2 = os.path.join(subdir, "other")
            os.mkdir(subdir2)
            # auditbeat_test/subdir/other/more.txt
            file2 = os.path.join(subdir2, "more.txt")
            self.create_file(file2, "")

            self.wait_log_contains("\"path\":\"{0}\"".format(escape_path(file2)), ignore_case=True)
            self.wait_output(4)
            self.wait_until(lambda: any(
                'file.path' in obj and obj['file.path'].lower() == subdir2.lower() for obj in self.read_output()))

            proc.check_kill_and_wait()
            self.assert_no_logged_warnings()

            # Ensure all Beater stages are used.
            assert self.log_contains("Setup Beat: auditbeat")
            assert self.log_contains("auditbeat start running")
            assert self.log_contains("auditbeat stopped")

            objs = self.read_output()

            has_file(objs, file1, "430ce34d020724ed75a196dfc2ad67c77772d169")
            has_file(objs, file2, "da39a3ee5e6b4b0d3255bfef95601890afd80709")
            has_dir(objs, subdir)
            has_dir(objs, subdir2)

            file_events(objs, file1, ['created'])
            file_events(objs, file2, ['created'])
