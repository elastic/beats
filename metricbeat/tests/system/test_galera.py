import os
import metricbeat
import unittest
from nose.plugins.attrib import attr

GALERA_FIELDS = metricbeat.COMMON_FIELDS + ["galera"]

GALERA_STATUS_FIELDS = ["local.state", "evs.state", "cluster.size", "cluster.status", "connected", "ready"]


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['galera']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_status(self):
        """
        Galera module outputs an event.
        """
        self.render_config_template(modules=[{
            "name": "galera",
            "metricsets": ["status"],
            "hosts": self.get_hosts(),
            "period": "5s",
            "query_mode": "small"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        self.assertItemsEqual(self.de_dot(GALERA_FIELDS), evt.keys())
        galera_info = evt["galera"]["status"]

        self.assert_fields_are_documented(evt)

    def get_hosts(self):
        return [os.getenv('GALERA_DSN', 'root:test@tcp(localhost:3306)/')]
