import os
import metricbeat
import unittest


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
            "path": "/test/metrics",
            "period": "1s",
            "namespace": "test",
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=10)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertTrue(len(output) >= 1)

    def get_hosts(self):
        return [os.getenv('DROPWIZARD_HOST', 'localhost') + ':' +
                os.getenv('DROPWIZARD_PORT', '8080')]
