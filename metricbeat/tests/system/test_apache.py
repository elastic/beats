import os
import metricbeat
import unittest
from nose.plugins.attrib import attr
import urllib2
import time

APACHE_FIELDS = metricbeat.COMMON_FIELDS + ["apache"]

APACHE_STATUS_FIELDS = ["hostname", "total_accesses", "total_kbytes",
                        "requests_per_sec", "bytes_per_sec", "bytes_per_request",
                        "workers.busy", "workers.idle", "uptime", "cpu",
                        "connections", "load", "scoreboard"]

CPU_FIELDS = ["load", "user", "system", "children_user",
              "children_system"]


class ApacheStatusTest(metricbeat.BaseTest):
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
        while found == False:
            res = urllib2.urlopen(hosts[0] + "/server-status?auto").read()
            if "CPULoad"  in res:
                found = True
            time.sleep(0.5)

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()

        # Ensure no errors or warnings exist in the log.
        log = self.get_log()
        self.assertNotRegexpMatches(log, "ERR|WARN")

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        # Verify the required fields are present.
        self.assertItemsEqual(self.de_dot(APACHE_FIELDS), evt.keys())
        apache_status = evt["apache"]["status"]
        self.assertItemsEqual(self.de_dot(APACHE_STATUS_FIELDS), apache_status.keys())
        self.assertItemsEqual(self.de_dot(CPU_FIELDS), apache_status["cpu"].keys())
        # There are more fields that could be checked.

        # Verify all fields present are documented.
        self.assert_fields_are_documented(evt)


    def get_hosts(self):
        return ['http://' + os.getenv('APACHE_HOST', 'localhost') + ':' +
                os.getenv('APACHE_PORT', '80')]
