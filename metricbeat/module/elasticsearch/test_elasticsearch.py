import re
import sys
import os
import unittest

sys.path.append(os.path.join(os.path.dirname(__file__), '../../tests/system'))

import metricbeat


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

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_index(self):
        """
        elasticsearch index metricset test
        """
        # TODO: need to create an index first to get stats
        self.check_metricset("elasticsearch", "index", self.get_hosts())

    def get_hosts(self):
        return [os.getenv('ES_HOST', 'localhost') + ':' +
                os.getenv('ES_PORT', '9200')]
