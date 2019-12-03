import os
import metricbeat
import unittest
from parameterized import parameterized

STAN_FIELDS = metricbeat.COMMON_FIELDS + ["stan"]


class TestNats(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['stan']

    @parameterized.expand([
        "stats",
        "channels",
        "subscriptions",
    ])
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_metricset(self, metricset):
        """
        etcd metricset tests
        """
        self.check_metricset("stan", metricset, self.get_hosts(), ['stan.' + metricset])
