import os
import metricbeat
import unittest
from nose.plugins.attrib import attr


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['munin']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_munin_node(self):
        namespace = "node_test"

        self.render_config_template(modules=[{
            "name": "munin",
            "metricsets": ["node"],
            "hosts": self.get_hosts(),
            "period": "1s",
            "extras": {
                "munin.plugins": ["cpu"],
            },
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=20)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertTrue(len(output) >= 1)
        evt = output[0]
        print(evt)

        assert evt["service"]["type"] == "cpu"
        assert evt["munin"]["plugin"]["name"] == "cpu"
        assert evt["munin"]["metrics"]["user"] > 0

        self.assert_fields_are_documented(evt)
