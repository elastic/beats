import os
import sys
import unittest
from xpack_metricbeat import XPackTest, metricbeat


class Test(XPackTest):
    COMPOSE_SERVICES = ['entsearch']
    COMPOSE_TIMEOUT = 600

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, 'integration test')
    def test_health(self):
        self.render_config_template(modules=[{
            "name": "enterprisesearch",
            "metricsets": ["health"],
            "hosts": [self.compose_host(service="enterprise_search")],
            "period": "5s"
        }])
        proc = self.start_beat(home=self.beat_path)
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]
        self.assert_fields_are_documented(evt)

        self.assertIn("enterprisesearch", evt)
        self.assertIn("health", evt["enterprisesearch"])

        entsearch_health = evt["enterprisesearch"]["health"]
        self.assertIn("jvm", entsearch_health)
