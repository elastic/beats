from xpack_metricbeat import XPackTest
import metricbeat
import unittest


class Test(XPackTest):
    COMPOSE_SERVICES = ['appsearch']
    COMPOSE_TIMEOUT = 600

    # This test timeouts in CI https://github.com/elastic/beats/issues/14057
    @unittest.skip("temporarily disabled")
    def test_stats(self):
        self.render_config_template(modules=[{
            "name": "appsearch",
            "metricsets": ["stats"],
            "hosts": [self.compose_host(service="appsearch")],
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

        self.assertIn("appsearch", evt)
        self.assertIn("stats", evt["appsearch"])

        appsearch_stats = evt["appsearch"]["stats"]
        self.assertIn("jvm", appsearch_stats)
        self.assertIn("queues", appsearch_stats)
