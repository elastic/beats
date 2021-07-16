import metricbeat
import os
import sys
import time
import unittest
from parameterized import parameterized


@metricbeat.parameterized_with_supported_versions
class Test(metricbeat.BaseTest):
    COMPOSE_SERVICES = ['pulsar']

    @parameterized.expand([
        "broker",
    ])
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_metricset(self, metricset):
        """
        pulsar metricset tests
        """
        self.check_metricset("pulsar", 'broker', self.get_hosts(), ['pulsar.' + 'broker'])
