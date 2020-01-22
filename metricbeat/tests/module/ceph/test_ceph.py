import os
import sys
import time
import unittest
from parameterized import parameterized

sys.path.append(os.path.join(os.path.dirname(__file__), '../../tests/system'))
import metricbeat


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['ceph']
    FIELDS = ["ceph"]

    @parameterized.expand([
        "cluster_disk",
        "cluster_health",
        "monitor_health",
        "pool_disk",
    ])
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_ceph(self, metricset):
        """
        ceph metricsets tests
        """
        self.check_metricset("ceph", metricset, self.get_hosts(), self.FIELDS)
