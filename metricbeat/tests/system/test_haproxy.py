import os
import metricbeat
import unittest
from nose.plugins.attrib import attr

HAPROXY_FIELDS = metricbeat.COMMON_FIELDS + ["haproxy"]

class Test(metricbeat.BaseTest):
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_info(self):
        """
        haproxy info metricset test
        """
        self.render_config_template(modules=[{
            "name": "haproxy",
            "metricsets": ["info"],
            "hosts": self.get_hosts(),
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()

        # Ensure no errors or warnings exist in the log.
        log = self.get_log()
        self.assertNotRegexpMatches(log.replace("WARN EXPERIMENTAL", ""), "ERR|WARN")

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]
        print evt

        self.assertItemsEqual(self.de_dot(HAPROXY_FIELDS), evt.keys(), evt)

        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_stat(self):
        """
        haproxy stat metricset test
        """
        self.render_config_template(modules=[{
            "name": "haproxy",
            "metricsets": ["stat"],
            "hosts": self.get_hosts(),
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()

        # TODO: There are lots of errors converting empty strings to numbers
        # during the schema conversion. This needs fixed.
        # Ensure no errors or warnings exist in the log.
        #log = self.get_log()
        #self.assertNotRegexpMatches(log.replace("WARN EXPERIMENTAL", ""), "ERR|WARN")

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            print evt
            self.assertItemsEqual(self.de_dot(HAPROXY_FIELDS), evt.keys(), evt)
            self.assert_fields_are_documented(evt)

    def get_hosts(self):
        return ["tcp://" + os.getenv('HAPROXY_HOST', 'localhost') + ':' +
                os.getenv('HAPROXY_PORT', '14567')]

