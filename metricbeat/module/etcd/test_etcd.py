import metricbeat
import os
import sys
import time
import unittest
from parameterized import parameterized


@metricbeat.parameterized_with_supported_versions
class Test(metricbeat.BaseTest):
    COMPOSE_SERVICES = ['etcd']

    @parameterized.expand([
        # "leader",
        "self",
        "store",
        "metrics",
    ])
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_metricset(self, metricset):
        """
        etcd metricset tests
        """
        self.check_metricset("etcd", metricset, self.get_hosts(), ['etcd.' + metricset])
