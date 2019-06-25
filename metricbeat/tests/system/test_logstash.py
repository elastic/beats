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

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_xpack(self):
        """
        logstash-xpack module tests
        """
        self.render_config_template(modules=[{
            "name": "logstash",
            "metricsets": ["node", "node_stats"],
            "hosts": self.get_hosts(),
            "period": "1s",
            "extras": {
                "xpack.enabled": "true"
            }
        }])

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()
