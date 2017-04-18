import os
import metricbeat
import unittest


class Test(metricbeat.BaseTest):

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_dropwizard(self):
        """
        dropwizard  metricset test
        """

        self.render_config_template(modules=[{
            "name": "dropwizard",
            "metricsets": ["collector"],
            "hosts": self.get_hosts(),
            "period": "1s",
            "namespace": "test",
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=20)
        proc.check_kill_and_wait()

        output = self.read_output_json()
        self.assertTrue(len(output) >= 1)

    def get_hosts(self):
        return [os.getenv('DROPWIZARD_HOST', 'localhost') + ':' +
                os.getenv('DROPWIZARD_PORT', '9090')]
