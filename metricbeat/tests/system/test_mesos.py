import os
import metricbeat
import unittest
from nose.plugins.attrib import attr

MESOS_FIELDS = metricbeat.COMMON_FIELDS + ["mesos"]


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['mesos']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_master(self):
        """
        Test mesos master metricset.
        """
        self.render_config_template(modules=[{
            "name": "mesos",
            "metricsets": ["master"],
            "hosts": self.get_hosts(),
            "period": "5s",
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=20)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertTrue(len(output) >= 1)
        event = output[0]

        fields = MESOS_FIELDS + ["master"]
        self.assertItemsEqual(self.de_dot(MESOS_FIELDS), event.keys())
        print(event)
        self.assertNotIn("error", event)

    def get_hosts(self):
        return [os.getenv('MESOS_HOST', 'localhost') + ':' +
                os.getenv('MESOS_PORT', '5050')]
