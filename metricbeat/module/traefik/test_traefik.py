import os
import sys
import unittest
import time
from parameterized import parameterized

sys.path.append(os.path.join(os.path.dirname(__file__), '../../tests/system'))

import metricbeat


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['traefik']
    FIELDS = ['traefik']

    @parameterized.expand([
        "health"
    ])
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_health(self, metricset):
        """
        traefik metricset tests
        """
        self.check_metricset("traefik", metricset, self.get_hosts(), self.FIELDS + ["service.name"])
