import jinja2
import os
import platform
import pytest
import sys
import time
import unittest

from auditbeat_xpack import *

COMMON_FIELDS = ["@timestamp", "host.name", "event.module", "event.dataset"]


class Test(AuditbeatXPackTest):

    def test_metricset_host(self):
        """
        host metricset collects general information about a server.
        """

        fields = ["host.name", "host.architecture", "host.hostname", "host.id", "host.ip", "host.mac",
                  "host.os.family", "host.os.kernel", "host.os.name", "host.os.platform", "host.os.version",
                  "system.audit.host.id", "system.audit.host.uptime", "system.audit.host.ip",
                  "system.audit.host.os.name"]

        self.check_metricset("system", "host", COMMON_FIELDS + fields)

    @unittest.skipUnless(sys.platform.startswith('linux'), "Only implemented for Linux")
    @unittest.skipIf(sys.byteorder != "little", "Test only implemented for little-endian systems")
    def test_metricset_login(self):
        """
        login metricset collects information about logins (successful and failed) and system restarts.
        """

        fields = ["event.origin", "event.outcome", "message", "process.pid", "source.ip",
                  "user.name", "user.terminal"]

        config = {
            "login.wtmp_file_pattern": os.path.abspath(os.path.join(self.beat_path, "module/system/login/testdata/wtmp*")),
            "login.btmp_file_pattern": os.path.abspath(os.path.join(self.beat_path, "module/system/login/testdata/btmp*")),
        }

        # Metricset is beta and that generates a warning, TODO: remove later
        self.check_metricset("system", "login", COMMON_FIELDS + fields, config, warnings_allowed=True)

    # 1/20 build fails https://github.com/elastic/beats/issues/21308
    @pytest.mark.flaky(reruns=1, reruns_delay=10)
    @unittest.skipIf(sys.platform == "win32", "Not implemented for Windows")
    @unittest.skipIf(sys.platform.startswith('linux') and not (os.path.isdir("/var/lib/dpkg") or os.path.isdir("/var/lib/rpm")),
                     "Only implemented for dpkg and rpm")
    @unittest.skipIf(sys.platform.startswith('darwin'), "See https://github.com/elastic/beats/issues/21308")
    def test_metricset_package(self):
        """
        package metricset collects information about installed packages on a system.
        """

        fields = ["system.audit.package.entity_id", "system.audit.package.name", "system.audit.package.version"]

        # Metricset is beta and that generates a warning, TODO: remove later
        self.check_metricset("system", "package", COMMON_FIELDS + fields, warnings_allowed=True)

    def test_metricset_process(self):
        """
        process metricset collects information about processes running on a system.
        """

        fields = ["process.entity_id", "process.pid", "process.parent.pid", "process.name", "process.executable",
                  "process.args", "process.start", "process.working_directory", "user.id", "user.group.id"]

        # Windows does not have effective and saved IDs, and user.name is not always filled for system processes.
        if sys.platform != "win32":
            fields.extend(["user.effective.id", "user.saved.id", "user.effective.group.id", "user.saved.group.id",
                           "user.name", "user.group.name"])

        # process.hash.max_file_size: 1 - To speed things up during testing, we effectively disable hashing.
        # errors_allowed|warnings_allowed=True - Disabling hashing causes the dataset to add an error to the event
        # and log a warning. That should not fail the test.
        self.check_metricset("system", "process", COMMON_FIELDS + fields, {"process.hash.max_file_size": 1},
                             errors_allowed=True, warnings_allowed=True)

    @unittest.skipUnless(sys.platform.startswith('linux'), "Only implemented for Linux")
    def test_metricset_user(self):
        """
        user metricset collects information about users on a server.
        """

        fields = ["user.entity_id", "system.audit.user.name"]

        self.check_metricset("system", "user", COMMON_FIELDS + fields)
