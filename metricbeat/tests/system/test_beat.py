import os
import metricbeat
import unittest
import time
from parameterized import parameterized
from elasticsearch import Elasticsearch


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['metricbeat', 'elasticsearch']
    FIELDS = ['beat']
    METRICSETS = ['stats', 'state']

    @parameterized.expand(METRICSETS)
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_metricsets(self, metricset):
        """
        beat metricset tests
        """
        self.check_metricset("beat", metricset, [self.compose_host("metricbeat")], self.FIELDS + ["service"])

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_xpack(self):
        """
        beat-xpack module tests
        """

        # Give the monitored Metricbeat instance enough time to collect metrics and index them
        # into Elasticsearch, so it may establish the connection to Elasticsearch and determine
        # it's cluster UUID in the process. Otherwise, the monitoring Metricbeat instance will
        # show errors in its log about not being able to determine the Elasticsearch cluster UUID
        # to be associated with the monitored Metricbeat instance.
        self.wait_until(cond=self.mb_connected_to_es, max_timeout=50)

        self.render_config_template(modules=[{
            "name": "beat",
            "metricsets": self.METRICSETS,
            "hosts": [self.compose_host("metricbeat")],
            "period": "1s",
            "extras": {
                "xpack.enabled": "true"
            }
        }])

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

    def mb_connected_to_es(self):
        return self.service_log_contains('metricbeat', 'Connection to backoff(elasticsearch(')
