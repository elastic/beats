import os
import metricbeat
import unittest
import time
from parameterized import parameterized


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['metricbeat']
    FIELDS = ['beat']

    @parameterized.expand([
        "stats",
        "state"
    ])
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_metricsets(self, metricset):
        """
        beat metricset tests
        """
        self.check_metricset("beat", metricset, self.get_hosts(), self.FIELDS + ["service"])
