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
        self.render_config_template(modules=[{
            "name": "beat",
            "metricsets": self.METRICSETS,
            "hosts": [self.compose_host("metricbeat")],
            "period": "1s",
            "extras": {
                "xpack.enabled": "true"
            }
        }])

        # Give the monitored Metricbeat instance enough time to collect metrics and index them
        # into Elasticsearch, so it may establish the connection to Elasticsearch and determine
        # it's cluster UUID in the process. Otherwise, the monitoring Metricbeat instance will
        # show errors in its log about not being able to determine the Elasticsearch cluster UUID
        # to be associated with the monitored Metricbeat instance.
        self.wait_until(cond=self.mb_index_exists, max_timeout=60)

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

    def mb_index_exists(self):
        es = Elasticsearch([self.get_elasticsearch_url()])
        return len(es.cat.indices(index='metricbeat-*')) > 0

    def get_elasticsearch_url(self):
        return "http://" + self.compose_host("elasticsearch")
