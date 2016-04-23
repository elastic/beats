import os
import metricbeat
from nose.plugins.attrib import attr

APACHE_FIELDS = metricbeat.COMMON_FIELDS + ["apache-status"]

APACHE_STATUS_FIELDS = ["hostname", "totalAccesses", "totalKBytes",
                        "reqPerSec", "bytesPerSec", "bytesPerReq",
                        "busyWorkers", "idleWorkers", "uptime", "cpu",
                        "connections", "load", "scoreboard"]

CPU_FIELDS = ["cpuLoad", "cpuUser", "cpuSystem", "cpuChildrenUser",
              "cpuChildrenSystem"]


class ApacheStatusTest(metricbeat.BaseTest):
    @attr('integration')
    def test_output(self):
        """
        Apache module outputs an event.
        """
        self.render_config_template(modules=[{
            "name": "apache",
            "metricsets": ["status"],
            "hosts": [os.getenv('APACHE_HOST')],
        }])
        proc = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=1)
        )
        proc.check_kill_and_wait()

        # Ensure no errors or warnings exist in the log.
        log = self.get_log()
        self.assertNotRegexpMatches(log, "ERR|WARN")

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        # Verify the required fields are present.
        self.assertItemsEqual(APACHE_FIELDS, evt.keys())
        apache_status = evt["apache-status"]
        self.assertItemsEqual(APACHE_STATUS_FIELDS, apache_status.keys())
        self.assertItemsEqual(CPU_FIELDS, apache_status["cpu"].keys())
        # There are more fields that could be checked.

        # Verify all fields present are documented.
        self.assert_fields_are_documented(evt)
