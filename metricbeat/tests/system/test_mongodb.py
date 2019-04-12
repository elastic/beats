import os
import metricbeat
import unittest
from nose.plugins.attrib import attr

MONGODB_FIELDS = metricbeat.COMMON_FIELDS + ["mongodb"]


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['mongodb']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_status(self):
        """
        MongoDB module outputs an event.
        """
        self.render_config_template(modules=[{
            "name": "mongodb",
            "metricsets": ["status"],
            "hosts": self.get_hosts(),
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        self.assertItemsEqual(self.de_dot(MONGODB_FIELDS + ["process"]), evt.keys())

        self.assert_fields_are_documented(evt)

    def get_hosts(self):
        return [os.getenv('MONGODB_HOST', 'localhost') + ':' +
                os.getenv('MONGODB_PORT', '27017')]
