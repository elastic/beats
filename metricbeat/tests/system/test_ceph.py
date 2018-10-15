import os
import metricbeat
import unittest
from parameterized import parameterized
import time


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

    def get_hosts(self):
        return [os.getenv('CEPH_HOST', 'localhost') + ':' +
                os.getenv('CEPH_PORT', '5000')]
