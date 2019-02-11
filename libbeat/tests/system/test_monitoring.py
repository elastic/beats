from base import BaseTest
import os
from elasticsearch import Elasticsearch
import re
from nose.plugins.attrib import attr
import unittest

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class Test(BaseTest):

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
            "Connection to .*elasticsearch\("+self.get_elasticsearch_url()+"\).* established")))
        self.wait_until(lambda: self.monitoring_doc_exists('beats_stats'))
        self.wait_until(lambda: self.monitoring_doc_exists('beats_state'))

        for monitoring_doc_type in ['beats_stats', 'beats_state']:
            field_names = ['cluster_uuid', 'timestamp', 'interval_ms', 'type', 'source_node', monitoring_doc_type]
            self.assert_monitoring_doc_contains_fields(monitoring_doc_type, field_names)

    def monitoring_doc_exists(self, monitoring_type):
        results = self.es_monitoring.search(
            index='.monitoring-beats-*',
            q='type:'+monitoring_type,
            size=1
        )
        hits = results['hits']['hits']
        return len(hits) == 1

    def assert_monitoring_doc_contains_fields(self, monitoring_type, field_names):
        results = self.es_monitoring.search(
            index='.monitoring-beats-*',
            q='type:'+monitoring_type,
            size=1
        )
        hits = results['hits']['hits']
        source = hits[0]['_source']

        for field_name in field_names:
            assert field_name in source

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
        self.es.cluster.put_settings(body={
            "transient": {
                "xpack.monitoring.collection.enabled": True
            }
        })

        # Delete any old beats monitoring data
        self.es_monitoring.indices.delete(index=".monitoring-beats-*", ignore=[404])

    def get_elasticsearch_monitoring_url(self):
        return "http://{host}:{port}".format(
            host=os.getenv("ES_MONITORING_HOST", "localhost"),
            port=os.getenv("ES_MONITORING_PORT", "9210")
        )
