import os
import sys
import unittest
from xpack_metricbeat import XPackTest, metricbeat


class Test(XPackTest):

    COMPOSE_SERVICES = ['openmetrics-node_exporter']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_openmetrics(self):
        """
        openmetrics collector test
        """
        self.render_config_template(modules=[{
            "name": "openmetrics",
            "metricsets": ["collector"],
            "hosts": self.get_hosts(),
            "period": "5s",
        }])
        proc = self.start_beat(home=self.beat_path)
        self.wait_until(lambda: self.output_lines() > 0, 60)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            self.assert_fields_are_documented(evt)
            self.assertIn("openmetrics", evt.keys(), evt)
            self.assertIn("metrics", evt["openmetrics"].keys(), evt)
            self.assertGreater(len(evt["openmetrics"]["metrics"].keys()), 0)
