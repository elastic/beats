import os
import sys
import unittest

sys.path.append(os.path.join(os.path.dirname(__file__), '../../tests/system'))
import metricbeat


class Test(metricbeat.BaseTest):

    FIELDS = ["aerospike"]
    COMPOSE_SERVICES = ['aerospike']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_namespace(self):
        """
        aerospike namespace metricset test
        """
        self.check_metricset("aerospike", "namespace", self.get_hosts(), self.FIELDS)
