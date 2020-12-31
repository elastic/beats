import metricbeat
import os
import sys
import unittest


@metricbeat.parameterized_with_supported_versions
class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['dropwizard']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_dropwizard(self):
        """
        dropwizard  metricset test
        """

        self.render_config_template(modules=[{
            "name": "dropwizard",
            "metricsets": ["collector"],
            "hosts": self.get_hosts(),
            "metrics_path": "/test/metrics",
            "period": "1s",
            "namespace": "test",
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=10)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertTrue(len(output) >= 1)
