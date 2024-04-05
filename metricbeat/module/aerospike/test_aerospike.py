import metricbeat
import unittest


class Test(metricbeat.BaseTest):

    FIELDS = ["aerospike"]
    COMPOSE_SERVICES = ['aerospike']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_namespace(self):
        """
        aerospike namespace metricset test
        """
        self.check_metricset("aerospike", "namespace", self.get_hosts(), self.FIELDS)
