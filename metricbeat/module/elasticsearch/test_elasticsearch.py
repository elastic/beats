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
        self.check_skip(metricset)

        self.start_trial()
        if metricset == "ml_job":
            self.create_ml_job()
        if metricset == "ccr":
            self.create_ccr_stats()

        es = Elasticsearch(self.get_hosts())
        es.indices.create(index='test-index', ignore=400)
        self.check_metricset("elasticsearch", metricset, self.get_hosts(), self.FIELDS +
                             ["service.name"], extras={"index_recovery.active_only": "false"})

    def get_hosts(self):
        return [os.getenv('ES_HOST', 'localhost') + ':' +
                os.getenv('ES_PORT', '9200')]

    def create_ml_job(self):
        es = Elasticsearch(self.get_hosts())

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

    def create_ccr_stats(self):
        self.setup_ccr_remote()
        self.create_ccr_leader_index()
        self.create_ccr_follower_index()

    def setup_ccr_remote(self):
        es = Elasticsearch(self.get_hosts())

        file = os.path.join(self.beat_path, "module", "elasticsearch", "ccr",
                            "_meta", "test", "test_remote_settings.json")

        body = {}
        with open(file, 'r') as f:
            body = json.load(f)

        path = "/_cluster/settings"
        es.transport.perform_request('PUT', path, body=body)

    def create_ccr_leader_index(self):
        es = Elasticsearch(self.get_hosts())

        file = os.path.join(self.beat_path, "module", "elasticsearch", "ccr", "_meta", "test", "test_leader_index.json")

        body = {}
        with open(file, 'r') as f:
            body = json.load(f)

        path = "/pied_piper"
        es.transport.perform_request('PUT', path, body=body)

    def create_ccr_follower_index(self):
        es = Elasticsearch(self.get_hosts())

        file = os.path.join(self.beat_path, "module", "elasticsearch", "ccr",
                            "_meta", "test", "test_follower_index.json")

        body = {}
        with open(file, 'r') as f:
            body = json.load(f)

        path = "/rats/_ccr/follow"
        es.transport.perform_request('PUT', path, body=body)

    def start_trial(self):
        es = Elasticsearch(self.get_hosts())

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

    def check_skip(self, metricset):
        if metricset != "ccr":
            return

        version = self.get_version()
        if semver.compare(version, "6.5.0") == -1:
            # Skip CCR metricset system test for Elasticsearch versions < 6.5.0 as CCR Stats
            # API endpoint is not available
            raise SkipTest("elasticsearch/ccr metricset system test only valid with Elasticsearch versions >= 6.5.0")

    def get_version(self):
        host = self.get_hosts()[0]
        res = urllib2.urlopen("http://" + host + "/").read()

        body = json.loads(res)
        version = body["version"]["number"]

        return version
