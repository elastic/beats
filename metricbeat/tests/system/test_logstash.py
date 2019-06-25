import os
import metricbeat
import unittest
import time


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['logstash']
    FIELDS = ['logstash']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_node(self):
        """
        logstash node metricset test
        """
        self.check_metricset("logstash", "node", self.get_hosts(), self.FIELDS + ["process"])

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_node_stats(self):
        """
        logstash node_stats metricset test
        """
        self.check_metricset("logstash", "node_stats", self.get_hosts(), self.FIELDS)
