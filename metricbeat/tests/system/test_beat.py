import os
import metricbeat
import unittest
import time
from parameterized import parameterized


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['metricbeat']
    FIELDS = ['beat']
    METRICSETS = ['stats', 'state']

    @parameterized.expand(METRICSETS)
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_metricsets(self, metricset):
        """
        beat metricset tests
        """
        self.check_metricset("beat", metricset, self.get_hosts(), self.FIELDS + ["service"])

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_xpack(self):
        """
        beat-xpack module tests
        """
        self.render_config_template(modules=[{
            "name": "beat",
            "metricsets": self.METRICSETS,
            "hosts": self.get_hosts(),
            "period": "1s",
            "extras": {
                "xpack.enabled": "true"
            }
        }])

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()
