import os
import unittest
from nose.plugins.attrib import attr
import urllib2
import time
import semver
import sys

sys.path.append(os.path.join(os.path.dirname(__file__), '../../tests/system'))
import metricbeat

APACHE_FIELDS = metricbeat.COMMON_FIELDS + ["apache"]

APACHE_STATUS_FIELDS = [
    "hostname", "total_accesses", "total_kbytes",
    "requests_per_sec", "bytes_per_sec", "bytes_per_request",
    "workers.busy", "workers.idle", "uptime", "cpu",
    "connections", "load", "scoreboard"
]

APACHE_OLD_STATUS_FIELDS = [
    "hostname", "total_accesses", "total_kbytes",
    "requests_per_sec", "bytes_per_sec", "bytes_per_request",
    "workers.busy", "workers.idle", "uptime", "cpu",
    "connections", "scoreboard"
]


CPU_FIELDS = [
    "load", "user", "system", "children_user", "children_system"
]


@metricbeat.parameterized_with_supported_versions
class ApacheStatusTest(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['apache']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_output(self):
        """
        Apache module outputs an event.
        """

        hosts = self.get_hosts()
        self.render_config_template(modules=[{
            "name": "apache",
            "metricsets": ["status"],
            "hosts": hosts,
            "period": "5s"
        }])

        found = False
        # Waits until CPULoad is part of the status
        while not found:
            res = urllib2.urlopen(hosts[0] + "/server-status?auto").read()
            if "CPULoad" in res:
                found = True
            time.sleep(0.5)

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        self.verify_fields(evt)

        # Verify all fields present are documented.
        self.assert_fields_are_documented(evt)

    def verify_fields(self, evt):
        self.assertItemsEqual(self.de_dot(APACHE_FIELDS), evt.keys())
        apache_status = evt["apache"]["status"]
        if self.old_apache_version():
            self.assertItemsEqual(
                self.de_dot(APACHE_OLD_STATUS_FIELDS), apache_status.keys())
        else:
            self.assertItemsEqual(
                self.de_dot(APACHE_STATUS_FIELDS), apache_status.keys())
            self.assertItemsEqual(
                self.de_dot(CPU_FIELDS), apache_status["cpu"].keys())
            # There are more fields that could be checked.

    def old_apache_version(self):
        if not 'APACHE_VERSION' in self.COMPOSE_ENV:
            return False

        version = self.COMPOSE_ENV['APACHE_VERSION']
        return semver.compare(version, '2.4.12') <= 0

    def get_hosts(self):
        return ['http://' + self.compose_host()]
