import os
import metricbeat
import unittest


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['elasticsearch']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_node(self):
        """
        elasticsearch node metricset test
        """
        self.check_metricset("elasticsearch", "node", self.get_hosts())

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_node_stats(self):
        """
        elasticsearch node_stats metricset test
        """
        self.check_metricset("elasticsearch", "node_stats", self.get_hosts())

    def get_hosts(self):
        return [os.getenv('ES_HOST', 'localhost') + ':' +
                os.getenv('ES_PORT', '9200')]
