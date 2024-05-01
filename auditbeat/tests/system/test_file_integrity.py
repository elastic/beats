import time
import unittest
import platform
from auditbeat import *
import platform
import time
import unittest

from auditbeat import *

if platform.platform().split('-')[0] == 'Linux':
    import pwd


def is_root():
    if 'geteuid' not in dir(os):
        return False
    return os.geteuid() == 0


def is_version_below(version, target):
    t = list(map(int, target.split('.')))
    v = list(map(int, version.split('.')))
    v += [0] * (len(t) - len(v))
    for i in range(len(t)):
        if v[i] != t[i]:
            return v[i] < t[i]
    return False


# Require Linux greater or equal than 3.10.0 and arm64/amd64 arch
def is_platform_supported():
    p = platform.platform().split('-')
    if p[0] != 'Linux':
        return False
    if is_version_below(p[1], '3.10.0'):
        return False
    return {'aarch64', 'arm64', 'x86_64', 'amd64'}.intersection(p)


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


def is_ubuntu_x86_64():
    if platform.system() == 'Ubuntu' in platform.platform():
        if platform.machine() == "x86_64":
            return True
    return False


def is_ubuntu_arm():
    if platform.system() == 'Ubuntu' in platform.platform():
        if platform.machine() == "aarch64":
            return True
    return False


class Test(BaseTest):
    def wait_output(self, min_events):
        self.wait_until(lambda: wrap_except(lambda: len(self.read_output()) >= min_events))
        # wait for the number of lines in the file to stay constant for 10 seconds
        prev_lines = -1
        while True:
            num_lines = self.output_lines()
            if prev_lines < num_lines:
                prev_lines = num_lines
                time.sleep(10)
            else:
                break

    def wait_startup(self, backend, dir):
        if backend == "ebpf":
            self.wait_log_contains("started ebpf watcher", max_timeout=30, ignore_case=True)
        if backend == "kprobes":
            self.wait_log_contains("Started kprobes watcher", max_timeout=30, ignore_case=True)
        else:
            # wait until the directories to watch are printed in the logs
            # this happens when the file_integrity module starts.
            # Case must be ignored under windows as capitalisation of paths
            # may differ
            self.wait_log_contains(escape_path(dir), max_timeout=30, ignore_case=True)

    def _assert_process_data(self, event, backend):
        if backend != "ebpf":
            return
        assert event["process.entity_id"] != ""
        assert event["process.executable"] == "pytest"
        assert event["process.pid"] == os.getpid()
        assert int(event["process.user.id"]) == os.geteuid()
        assert event["process.user.name"] == pwd.getpwuid(os.geteuid()).pw_name
        assert int(event["process.group.id"]) == os.getegid()

    def _test_non_recursive(self, backend):
        """
        file_integrity monitors watched directories (non recursive).
        """

        dirs = [self.temp_dir("auditbeat_test"),
                self.temp_dir("auditbeat_test")]

        with PathCleanup(dirs):
            extras = {
                "paths": dirs,
                "scan_at_start": False
            }
            if platform.system() == "Linux":
                extras["backend"] = backend

            self.render_config_template(
                modules=[{
                    "name": "file_integrity",
                    "extras": extras
                }],
            )
            proc = self.start_beat()
            self.wait_startup(backend, dirs[0])

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

            if backend == "fsnotify" or backend == "kprobes":
                self.wait_output(4)
            else:
                # ebpf backend doesn't catch directory creation
                self.wait_output(3)

            proc.check_kill_and_wait()
            self.assert_no_logged_warnings()

            # Ensure all Beater stages are used.
            assert self.log_contains("Setup Beat: auditbeat")
            assert self.log_contains("auditbeat start running")
            assert self.log_contains("auditbeat stopped")

            objs = self.read_output()

            has_file(objs, file1, "430ce34d020724ed75a196dfc2ad67c77772d169")
            has_file(objs, file2, "d23be250530a24be33069572db67995f21244c51")
            if backend == "fsnotify" or backend == "kprobes":
                has_dir(objs, subdir)

            file_events(objs, file1, ['created', 'deleted'])
            file_events(objs, file2, ['created'])

            # assert file inside subdir is not reported
            assert self.log_contains(file3) is False

            self._assert_process_data(objs[0], backend)

    @unittest.skipIf(os.getenv("CI") is not None and platform.system() == 'Darwin',
                     'Flaky test: https://github.com/elastic/beats/issues/24678')
    def test_non_recursive__fsnotify(self):
        self._test_non_recursive("fsnotify")

    @unittest.skipUnless(is_root(), "Requires root")
    @unittest.skipIf(is_ubuntu_x86_64(), "Timeout error on Buildkite - https://github.com/elastic/ingest-dev/issues/3270")
    @unittest.skipIf(is_ubuntu_arm(), "Timeout error on Buildkite - https://github.com/elastic/ingest-dev/issues/3270")
    def test_non_recursive__ebpf(self):
        self._test_non_recursive("ebpf")

    @unittest.skipUnless(is_platform_supported(), "Requires Linux 3.10.0+ and arm64/amd64 arch")
    @unittest.skipUnless(is_root(), "Requires root")
    @unittest.skipIf(is_ubuntu_x86_64(), "Timeout error on Buildkite - https://github.com/elastic/ingest-dev/issues/3270")
    def test_non_recursive__kprobes(self):
        self._test_non_recursive("kprobes")

    def _test_recursive(self, backend):
        """
        file_integrity monitors watched directories (recursive).
        """

        dirs = [self.temp_dir("auditbeat_test")]

        with PathCleanup(dirs):
            extras = {
                "paths": dirs,
                "scan_at_start": False,
                "recursive": True
            }
            if platform.system() == "Linux":
                extras["backend"] = backend

            self.render_config_template(
                modules=[{
                    "name": "file_integrity",
                    "extras": extras
                }],
            )
            proc = self.start_beat()
            self.wait_startup(backend, dirs[0])

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

            if backend == "fsnotify" or backend == "kprobes":
                self.wait_output(4)
                self.wait_until(lambda: any(
                    'file.path' in obj and obj['file.path'].lower() == subdir2.lower() for obj in self.read_output()))
            else:
                # ebpf backend doesn't catch directory creation
                self.wait_output(2)

            proc.check_kill_and_wait()
            self.assert_no_logged_warnings()

            # Ensure all Beater stages are used.
            assert self.log_contains("Setup Beat: auditbeat")
            assert self.log_contains("auditbeat start running")
            assert self.log_contains("auditbeat stopped")

            objs = self.read_output()

            has_file(objs, file1, "430ce34d020724ed75a196dfc2ad67c77772d169")
            has_file(objs, file2, "da39a3ee5e6b4b0d3255bfef95601890afd80709")
            if backend == "fsnotify" or backend == "kprobes":
                has_dir(objs, subdir)
                has_dir(objs, subdir2)

            file_events(objs, file1, ['created'])
            file_events(objs, file2, ['created'])

            self._assert_process_data(objs[0], backend)

    def test_recursive__fsnotify(self):
        self._test_recursive("fsnotify")

    @unittest.skipUnless(is_root(), "Requires root")
    @unittest.skipIf(is_ubuntu_x86_64(), "Timeout error on Buildkite - https://github.com/elastic/ingest-dev/issues/3270")
    @unittest.skipIf(is_ubuntu_arm(), "Timeout error on Buildkite - https://github.com/elastic/ingest-dev/issues/3270")
    def test_recursive__ebpf(self):
        self._test_recursive("ebpf")

    @unittest.skipUnless(is_platform_supported(), "Requires Linux 3.10.0+ and arm64/amd64 arch")
    @unittest.skipUnless(is_root(), "Requires root")
    @unittest.skipIf(is_ubuntu_x86_64, "Timeout error on Buildkite - https://github.com/elastic/ingest-dev/issues/3270")
    def test_recursive__kprobes(self):
        self._test_recursive("kprobes")

    @unittest.skipIf(platform.system() != 'Linux', 'Non linux, skipping.')
    def _test_file_modified(self, backend):
        """
        file_integrity tests for file modifications (chmod, chown, write, truncate, xattrs).
        """

        dirs = [self.temp_dir("auditbeat_test")]

        with PathCleanup(dirs):
            self.render_config_template(
                modules=[{
                    "name": "file_integrity",
                    "extras": {
                        "paths": dirs,
                        "scan_at_start": False,
                        "recursive": False,
                        "backend": backend
                    }
                }],
            )
            proc = self.start_beat()
            self.wait_startup(backend, dirs[0])

            # Event 1: file create
            f = os.path.join(dirs[0], f'file_{backend}.txt')
            self.create_file(f, "hello world!")

            # FSNotify can't catch the events if operations happens too fast
            time.sleep(1)

            # Event 2: chmod
            os.chmod(f, 0o777)
            # FSNotify can't catch the events if operations happens too fast
            time.sleep(1)

            with open(f, "w") as fd:
                # Event 3: write
                fd.write("data")
                # FSNotify can't catch the events if operations happens too fast
                time.sleep(1)

                # Event 4: truncate
                fd.truncate(0)
                # FSNotify can't catch the events if operations happens too fast
                time.sleep(1)

            # Wait N events
            self.wait_output(4)

            proc.check_kill_and_wait()
            self.assert_no_logged_warnings()

            # Ensure all Beater stages are used.
            assert self.log_contains("Setup Beat: auditbeat")
            assert self.log_contains("auditbeat start running")
            assert self.log_contains("auditbeat stopped")

    @unittest.skipIf(platform.system() != 'Linux', 'Non linux, skipping.')
    def test_file_modified__fsnotify(self):
        self._test_file_modified("fsnotify")

    @unittest.skipIf(platform.system() != 'Linux', 'Non linux, skipping.')
    @unittest.skipUnless(is_root(), "Requires root")
    def test_file_modified__ebpf(self):
        self._test_file_modified("ebpf")

    @unittest.skipUnless(is_platform_supported(), "Requires Linux 3.10.0+ and arm64/amd64 arch")
    @unittest.skipUnless(is_root(), "Requires root")
    @unittest.skipIf(is_ubuntu_x86_64, "Timeout error on Buildkite - https://github.com/elastic/ingest-dev/issues/3270")
    def test_file_modified__kprobes(self):
        self._test_file_modified("kprobes")
