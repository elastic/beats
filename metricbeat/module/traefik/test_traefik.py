import os
import metricbeat
import unittest
import time
from parameterized import parameterized


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['traefik']

    @parameterized.expand([
        "health"
    ])
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_health(self, metricset):
        """
        traefik metricset tests
        """
        self.check_metricset("traefik", metricset, self.get_hosts(), self.FIELDS + ["service.name"])

    def get_hosts(self):
        return [os.getenv('TRAEFIK_HOST', 'localhost') + ':' +
                os.getenv('TRAEFIK_API_PORT', '8080')]
