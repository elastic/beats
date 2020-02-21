import os
import redis
import sys
import unittest
from nose.plugins.attrib import attr

sys.path.append(os.path.join(os.path.dirname(__file__), '../../tests/system'))
import metricbeat


@metricbeat.parameterized_with_supported_versions
class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['redis']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @parameterized.expand([
        "node",
        "proxy"
    ])
    @attr('integration')
    def test_metricset(self, metricset):
        """
        Test redis enterprise metricset
        """

        if self.oss_distribution():
            self.skipTest("only oss distribution is supported")
            return

        self.render_config_template(modules=[{
            "name": "redis",
            "metricsets": [metricset],
            "hosts": ['https://' + self.compose_host(port='8070/tcp')],
            "period": "5s",
            "ssl.verification_mode": "none"
        }])
        proc = self.start_beat(home=self.beat_path)
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            self.assert_fields_are_documented(evt)
            self.assertIn("prometheus", evt.keys(), evt)
            self.assertIn("metrics", evt["prometheus"].keys(), evt)
            self.assertGreater(len(evt["prometheus"]["metrics"].keys()), 0)

            # Verify if processors are correctly setup.
            for metric in evt["prometheus"]["metrics"].keys():
                assert metric.startswith(metricset + "_")

    def oss_distribution(self):
        if not 'REDIS_DISTRIBUTION' in self.COMPOSE_ENV:
            return False

        return self.COMPOSE_ENV['REDIS_DISTRIBUTION'] == 'oss'
