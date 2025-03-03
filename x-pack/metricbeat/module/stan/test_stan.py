import os
import sys
import unittest
from parameterized import parameterized
from xpack_metricbeat import XPackTest, metricbeat

STAN_FIELDS = metricbeat.COMMON_FIELDS + ["stan"]


@unittest.skip("tests failing: https://github.com/elastic/beats/issues/34844")
class TestStan(XPackTest):

    COMPOSE_SERVICES = ['stan']

    @parameterized.expand([
        "stats",
        "channels",
        "subscriptions",
    ])
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_metricset(self, metricset):
        """
        stan metricset tests
        """
        self.check_metricset("stan", metricset, self.get_hosts(), STAN_FIELDS)
