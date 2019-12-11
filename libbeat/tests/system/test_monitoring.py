from base import BaseTest
import os
from elasticsearch import Elasticsearch
import re
from nose.plugins.attrib import attr
import unittest
import requests
import random
import string

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

        self.clean_output_cluster()
        self.clean_monitoring_cluster()
        self.init_output_cluster()

        proc = self.start_beat(config="mockbeat.yml")
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains(re.compile("\[monitoring\].*Publish event")))
        self.wait_until(lambda: self.log_contains(re.compile(
            "Connection to .*elasticsearch\("+self.get_elasticsearch_url()+"\).* established")))
        self.wait_until(lambda: self.monitoring_doc_exists('beats_stats'))
        self.wait_until(lambda: self.monitoring_doc_exists('beats_state'))

        proc.check_kill_and_wait()

        for monitoring_doc_type in ['beats_stats', 'beats_state']:
            field_names = ['cluster_uuid', 'timestamp', 'interval_ms', 'type', 'source_node', monitoring_doc_type]
            self.assert_monitoring_doc_contains_fields(monitoring_doc_type, field_names)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_direct_to_monitoring_cluster(self):
        """
        Test shipping monitoring data directly to the monitoring cluster.
        Make sure expected documents are indexed in monitoring cluster.
        """

        self.render_config_template(
            "mockbeat",
            monitoring={
                "elasticsearch": {
                    "hosts": [self.get_elasticsearch_monitoring_url()]
                }
            }
        )

        self.clean_output_cluster()
        self.clean_monitoring_cluster()

        proc = self.start_beat(config="mockbeat.yml")
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains(re.compile("\[monitoring\].*Publish event")))
        self.wait_until(lambda: self.log_contains(re.compile(
            "Connection to .*elasticsearch\("+self.get_elasticsearch_monitoring_url()+"\).* established")))
        self.wait_until(lambda: self.monitoring_doc_exists('beats_stats'))
        self.wait_until(lambda: self.monitoring_doc_exists('beats_state'))

        proc.check_kill_and_wait()

        for monitoring_doc_type in ['beats_stats', 'beats_state']:
            field_names = ['cluster_uuid', 'timestamp', 'interval_ms', 'type', monitoring_doc_type]
            self.assert_monitoring_doc_contains_fields(monitoring_doc_type, field_names)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_compare(self):
        """
        Test that monitoring docs are the same, regardless of how they are shipped.
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

        self.clean_output_cluster()
        self.clean_monitoring_cluster()
        self.init_output_cluster()

        proc = self.start_beat(config="mockbeat.yml")
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains(re.compile("\[monitoring\].*Publish event")))
        self.wait_until(lambda: self.log_contains(re.compile(
            "Connection to .*elasticsearch\("+self.get_elasticsearch_url()+"\).* established")))
        self.wait_until(lambda: self.monitoring_doc_exists('beats_stats'))
        self.wait_until(lambda: self.monitoring_doc_exists('beats_state'))

        proc.check_kill_and_wait()

        indirect_beats_stats_doc = self.get_monitoring_doc('beats_stats')
        indirect_beats_state_doc = self.get_monitoring_doc('beats_state')

        self.render_config_template(
            "mockbeat",
            monitoring={
                "elasticsearch": {
                    "hosts": [self.get_elasticsearch_monitoring_url()]
                }
            }
        )

        self.clean_output_cluster()
        self.clean_monitoring_cluster()

        proc = self.start_beat(config="mockbeat.yml")
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains(re.compile("\[monitoring\].*Publish event")))
        self.wait_until(lambda: self.log_contains(re.compile(
            "Connection to .*elasticsearch\("+self.get_elasticsearch_monitoring_url()+"\).* established")))
        self.wait_until(lambda: self.monitoring_doc_exists('beats_stats'))
        self.wait_until(lambda: self.monitoring_doc_exists('beats_state'))

        proc.check_kill_and_wait()

        direct_beats_stats_doc = self.get_monitoring_doc('beats_stats')
        direct_beats_state_doc = self.get_monitoring_doc('beats_state')

        self.assert_same_structure(indirect_beats_state_doc['beats_state'], direct_beats_state_doc['beats_state'])
        self.assert_same_structure(indirect_beats_stats_doc['beats_stats'], direct_beats_stats_doc['beats_stats'])

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_cluster_uuid_setting(self):
        """
        Test that monitoring.cluster_uuid setting may be set without any other monitoring.* settings
        """
        test_cluster_uuid = self.random_string(10)
        self.render_config_template(
            "mockbeat",
            monitoring={
                "cluster_uuid": test_cluster_uuid
            },
            http_enabled="true"
        )

        proc = self.start_beat(config="mockbeat.yml")
        self.wait_until(lambda: self.log_contains("mockbeat start running."))

        state = self.get_beat_state()
        proc.check_kill_and_wait()

        self.assertEqual(test_cluster_uuid, state["monitoring"]["cluster_uuid"])

    def search_monitoring_doc(self, monitoring_type):
        results = self.es_monitoring.search(
            index='.monitoring-beats-*',
            q='type:'+monitoring_type,
            size=1
        )
        return results['hits']['hits']

    def monitoring_doc_exists(self, monitoring_type):
        hits = self.search_monitoring_doc(monitoring_type)
        return len(hits) == 1

    def get_monitoring_doc(self, monitoring_type):
        hits = self.search_monitoring_doc(monitoring_type)
        if len(hits) != 1:
            return None
        return hits[0]['_source']

    def assert_monitoring_doc_contains_fields(self, monitoring_type, field_names):
        results = self.es_monitoring.search(
            index='.monitoring-beats-*',
            q='type:'+monitoring_type,
            size=1
        )
        hits = results['hits']['hits']
        source = hits[0]['_source']

        for field_name in field_names:
            self.assertIn(field_name, source)

    def assert_same_structure(self, dict1, dict2):
        dict1_keys = dict1.keys()
        dict2_keys = dict2.keys()

        self.assertEqual(len(dict1_keys), len(dict2_keys))
        for key in dict1_keys:
            dict1_val = dict1[key]
            dict2_val = dict2[key]

            # Cast ints to floats for more practical type comparison further down
            if isinstance(dict1_val, int):
                dict1_val = float(dict1_val)
            if isinstance(dict2_val, int):
                dict2_val = float(dict2_val)
            self.assertEqual(type(dict1_val), type(dict2_val))

            if isinstance(dict1_val, dict):
                self.assert_same_structure(dict1_val, dict2_val)

    def clean_output_cluster(self):
        # Remove all exporters
        self.es.cluster.put_settings(body={
            "transient": {
                "xpack.monitoring.exporters.*": None
            }
        })

        # Disable collection
        self.es.cluster.put_settings(body={
            "transient": {
                "xpack.monitoring.collection.enabled": None
            }
        })

    def clean_monitoring_cluster(self):
        # Delete any old beats monitoring data
        self.es_monitoring.indices.delete(index=".monitoring-beats-*", ignore=[404])

    def init_output_cluster(self):
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

    def get_elasticsearch_monitoring_url(self):
        return "http://{host}:{port}".format(
            host=os.getenv("ES_MONITORING_HOST", "localhost"),
            port=os.getenv("ES_MONITORING_PORT", "9210")
        )

    def get_beat_state(self):
        url = "http://localhost:5066/state"
        return requests.get(url).json()

    def random_string(self, size):
        char_pool = string.ascii_letters + string.digits
        return ''.join(random.choice(char_pool) for i in range(size))
