import re
import sys
import os
import unittest
from elasticsearch import Elasticsearch, TransportError
from parameterized import parameterized
from nose.plugins.skip import SkipTest
import urllib2
import json
import semver

sys.path.append(os.path.join(os.path.dirname(__file__), '../../tests/system'))

import metricbeat


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['elasticsearch']
    FIELDS = ["elasticsearch"]

    @parameterized.expand([
        "ccr",
        "index",
        "index_summary",
        "ml_job",
        "index_recovery",
        "node_stats",
        "node",
        "shard"
    ])
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_metricsets(self, metricset):
        """
        elasticsearch metricset tests
        """
        es = Elasticsearch(self.get_hosts())
        self.check_skip(metricset, es)

        self.start_trial(es)
        if metricset == "ml_job":
            self.create_ml_job(es)
        if metricset == "ccr":
            self.create_ccr_stats(es)

        es.indices.create(index='test-index', ignore=400)
        self.check_metricset("elasticsearch", metricset, self.get_hosts(), self.FIELDS +
                             ["service.name"], extras={"index_recovery.active_only": "false"})

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_xpack_cluster_stats(self):
        """
        elasticsearch-xpack module test for type:cluster_stats
        """

        es = Elasticsearch(self.get_hosts())
        self.start_basic(es)

        self.render_config_template(modules=[{
            "name": "elasticsearch",
            "metricsets": [
                "ccr",
                "cluster_stats",
                "index",
                "index_recovery",
                "index_summary",
                "ml_job",
                "node_stats",
                "shard"
            ],
            "hosts": self.get_hosts(),
            "period": "1s",
            "extras": {
                "xpack.enabled": "true"
            }
        }])
        proc = self.start_beat()
        self.wait_log_contains('"type": "cluster_stats"')

        # self.wait_until(lambda: self.output_has_message('"type":"cluster_stats"'))
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        docs = self.read_output_json()
        for doc in docs:
            if not "type" in doc:
                continue
            t = doc["type"]
            if t != "cluster_stats":
                continue
            license = doc["license"]
            issue_date = license["issue_date_in_millis"]
            self.assertIsNot(type(issue_date), float)

            self.assertNotIn("expiry_date_in_millis", license)

    def get_hosts(self):
        return [os.getenv('ES_HOST', 'localhost') + ':' +
                os.getenv('ES_PORT', '9200')]

    def create_ml_job(self, es):
        # Check if an ml job already exists
        response = es.transport.perform_request('GET', "/_xpack/ml/anomaly_detectors/_all/")
        if response["count"] > 0:
            return

        file = os.path.join(self.beat_path, "module", "elasticsearch", "ml_job", "_meta", "test", "test_job.json")

        body = {}
        with open(file, 'r') as f:
            body = json.load(f)

        path = "/_xpack/ml/anomaly_detectors/test"
        es.transport.perform_request('PUT', path, body=body)

    def create_ccr_stats(self, es):
        self.setup_ccr_remote(es)
        self.create_ccr_leader_index(es)
        self.create_ccr_follower_index(es)

    def setup_ccr_remote(self, es):
        file = os.path.join(self.beat_path, "module", "elasticsearch", "ccr",
                            "_meta", "test", "test_remote_settings.json")

        body = {}
        with open(file, 'r') as f:
            body = json.load(f)

        path = "/_cluster/settings"
        es.transport.perform_request('PUT', path, body=body)

    def create_ccr_leader_index(self, es):
        file = os.path.join(self.beat_path, "module", "elasticsearch", "ccr", "_meta", "test", "test_leader_index.json")

        body = {}
        with open(file, 'r') as f:
            body = json.load(f)

        path = "/pied_piper"
        es.transport.perform_request('PUT', path, body=body)

    def create_ccr_follower_index(self, es):
        file = os.path.join(self.beat_path, "module", "elasticsearch", "ccr",
                            "_meta", "test", "test_follower_index.json")

        body = {}
        with open(file, 'r') as f:
            body = json.load(f)

        path = "/rats/_ccr/follow"
        es.transport.perform_request('PUT', path, body=body)

    def start_trial(self, es):
        # Check if trial is already enabled
        response = es.transport.perform_request('GET', "/_xpack/license")
        if response["license"]["type"] == "trial":
            return

        # Enable xpack trial
        try:
            es.transport.perform_request('POST', "/_xpack/license/start_trial?acknowledge=true")
        except:
            e = sys.exc_info()[0]
            print "Trial already enabled. Error: {}".format(e)

    def start_basic(self, es):
        # Check if basic license is already enabled
        response = es.transport.perform_request('GET', "/_xpack/license")
        if response["license"]["type"] == "basic":
            return

        try:
            es.transport.perform_request('POST', "/_xpack/license/start_basic?acknowledge=true")
        except:
            e = sys.exc_info()[0]
            print "Basic license already enabled. Error: {}".format(e)

    def check_skip(self, metricset, es):
        if metricset != "ccr":
            return

        version = self.get_version(es)
        if semver.compare(version, "6.5.0") == -1:
            # Skip CCR metricset system test for Elasticsearch versions < 6.5.0 as CCR Stats
            # API endpoint is not available
            raise SkipTest("elasticsearch/ccr metricset system test only valid with Elasticsearch versions >= 6.5.0")

    def get_version(self, es):
        response = es.transport.perform_request('GET', "/")
        return response["version"]["number"]
