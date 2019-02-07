from base import BaseTest
import os
from elasticsearch import Elasticsearch
import re
from nose.plugins.attrib import attr
import unittest

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class Test(BaseTest):

    # production cluster: 9200
    # monitoring cluster: 9210
    def setUp(self):
        super(BaseTest, self).setUp()

        self.es = Elasticsearch([self.get_elasticsearch_url()])
        self.es_monitoring = Elasticsearch([self.get_elasticsearch_monitoring_url()])

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_via_output_cluster(self):
        """
        Test shipping monitoring data via the elasticsearch output cluster.
        Make sure expected documents are indexed in monitoring cluster.
        """

        self.render_config_template(
            "mockbeat",
            xpack={
                "monitoring": {
                    "elasticsearch": {
                        "hosts": [self.get_elasticsearch_url()]
                    }
                }
            }
        )

        self.clean()

        proc = self.start_beat(config="mockbeat.yml")
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains(re.compile("\[monitoring\].*Publish event")))
        self.wait_until(lambda: self.log_contains(re.compile(
            "Connection to .*elasticsearch\(http://localhost:9200\).* established")))

    def clean(self):
        # Setup remote exporter
        self.es.cluster.put_settings(body={
            "transient": {
                "xpack.monitoring.exporters.my_remote": {
                    "type": "http",
                    "host": [self.get_elasticsearch_monitoring_url()]
                }
            }
        })

        # Enable collection
        # try:
        self.es.cluster.put_settings(body={
            "transient": {
                "xpack.monitoring.collection.enabled": True
            }
        })

        # Delete any old beats monitoring data
        # try:
        self.es_monitoring.indices.delete(index=".monitoring-beats-*", ignore=[404])
        # except:
        #     pass

    def get_elasticsearch_monitoring_url(self):
        return "http://{host}:{port}".format(
            host=os.getenv("ES_MONITORING_HOST", "localhost"),
            port=os.getenv("ES_MONITORING_PORT", "9210")
        )
