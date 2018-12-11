import jinja2
import os
import sys
import time
import unittest

sys.path.append(os.path.join(os.path.dirname(__file__), '../../../../auditbeat/tests/system'))

from auditbeat_xpack import *

COMMON_FIELDS = ["@timestamp", "host.name", "event.module", "event.dataset"]


class Test(AuditbeatXPackTest):

    def test_metricset_host(self):
        """
        host metricset collects general information about a server.
        """

        fields = ["system.host.uptime", "system.host.ip", "system.host.os.name"]

        # Metricset is experimental and that generates a warning, TODO: remove later
        self.check_metricset("system", "host", COMMON_FIELDS + fields, warnings_allowed=True)

    def test_metricset_packages(self):
        """
        packages metricset collects information about installed packages on a system.
        """

        fields = ["system.packages.package"]

        # Metricset is experimental and that generates a warning, TODO: remove later
        self.check_metricset("system", "packages", COMMON_FIELDS + fields, warnings_allowed=True)

    @unittest.skipIf(sys.platform == "darwin" and os.geteuid != 0, "Requires root on macOS")
    @unittest.skip("Test is failing on CI")
    def test_metricset_process(self):
        """
        process metricset collects information about processes running on a system.
        """

        fields = ["process.pid"]

        # Metricset is experimental and that generates a warning, TODO: remove later
        # TODO: Remove try/catch once new fields are in fields.ecs.yml
        try:
            self.check_metricset("system", "process", COMMON_FIELDS + fields, warnings_allowed=True)
        except Exception as e:
            if "process.working_directory" not in str(e) and "process.start" not in str(e):
                raise

    @unittest.skipUnless(sys.platform == "linux2", "Only implemented for Linux")
    def test_metricset_socket(self):
        """
        socket metricset collects information about open sockets on a system.
        """

        fields = ["destination.port"]

        # Metricset is experimental and that generates a warning, TODO: remove later
        # TODO: Remove try/catch once `network.type` is in fields.ecs.yml
        try:
            self.check_metricset("system", "socket", COMMON_FIELDS + fields, warnings_allowed=True)
        except Exception as e:
            if "network.type" not in str(e):
                raise

    @unittest.skipUnless(sys.platform == "linux2", "Only implemented for Linux")
    def test_metricset_user(self):
        """
        user metricset collects information about users on a server.
        """

        fields = ["system.user.name"]

        # Metricset is experimental and that generates a warning, TODO: remove later
        self.check_metricset("system", "user", COMMON_FIELDS + fields, warnings_allowed=True)
