import metricbeat
import os
import sys
import time
import unittest
from parameterized import parameterized


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
