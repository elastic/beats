import jinja2
import os
import sys
import time

sys.path.append(os.path.join(os.path.dirname(__file__), '../../../../auditbeat/tests/system'))

from auditbeat_xpack import *

COMMON_FIELDS = ["@timestamp", "beat.version", "host.name", "event.module", "event.dataset"]

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

        fields = ["system.packages.packages.package.name"]
        
        # Metricset is experimental and that generates a warning, TODO: remove later
        self.check_metricset("system", "packages", COMMON_FIELDS + fields, warnings_allowed=True)

    def test_metricset_processes(self):
        """
        processes metricset collects information about processes running on a system.
        """

        fields = ["system.processes.processes.process.name"]
        
        # Metricset is experimental and that generates a warning, TODO: remove later
        self.check_metricset("system", "processes", COMMON_FIELDS + fields, warnings_allowed=True)
