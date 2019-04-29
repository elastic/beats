import os
import sys
import unittest
import time
from parameterized import parameterized

sys.path.append(os.path.join(os.path.dirname(__file__), '../../tests/system'))

import metricbeat


class Test(metricbeat.BaseTest):
    COMPOSE_SERVICES = ['etcd']

    @parameterized.expand([
        "leader",
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


class Test_3_2(Test):
    COMPOSE_SERVICES = ['etcd_3_2']
