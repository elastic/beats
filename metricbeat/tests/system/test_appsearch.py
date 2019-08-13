import metricbeat
import unittest


class Test(metricbeat.BaseTest):
    COMPOSE_SERVICES = ['elasticsearch', 'appsearch']
    COMPOSE_TIMEOUT = 600

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_status(self):
        self.render_config_template(modules=[{
            "name": "appsearch",
            "metricsets": ["stats"],
            "hosts": ["localhost:3002"],
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]
        self.assert_fields_are_documented(evt)

        self.assertIn("appsearch", evt)
        self.assertIn("stats", evt["appsearch"])

        appsearch_stats = evt["appsearch"]["stats"]
        self.assertIn("jvm", appsearch_stats)
        self.assertIn("queues", appsearch_stats)
