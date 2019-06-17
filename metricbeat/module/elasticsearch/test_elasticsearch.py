import re
import sys
import os
import unittest
from elasticsearch import Elasticsearch, TransportError, client
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

    def setUp(self):
        super(Test, self).setUp()
        self.es = Elasticsearch(self.get_hosts())
        self.ml_es = client.xpack.ml.MlClient(self.es)

        es_version = self.get_version()
        if es_version["major"] < 7:
            self.license_url = "/_xpack/license"
            self.ml_anomaly_detectors_url = "/_xpack/ml/anomaly_detectors"
        else:
            self.license_url = "/_license"
            self.ml_anomaly_detectors_url = "/_ml/anomaly_detectors"

        self.start_trial()
        self.es.indices.create(index='test_index', ignore=400)

    def tearDown(self):
        self.ccr_unfollow_index()
        self.es.indices.delete(index='test_index,pied_piper,rats', ignore_unavailable=True)
        self.delete_ml_job()
        super(Test, self).tearDown()

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

        if metricset == "ml_job":
            self.create_ml_job()
        if metricset == "ccr":
            self.create_ccr_stats()

        self.check_metricset("elasticsearch", metricset, self.get_hosts(), self.FIELDS +
                             ["service"], extras={"index_recovery.active_only": "false"})

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_xpack(self):
        """
        elasticsearch-xpack module tests
        """
        es = Elasticsearch(self.get_hosts())

        self.create_ml_job()
        self.create_ccr_stats()

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
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

    def create_ml_job(self):
        # Check if an ml job already exists
        response = self.ml_es.get_jobs()
        if response["count"] > 0:
            return

        file = os.path.join(self.beat_path, "module", "elasticsearch", "ml_job", "_meta", "test", "test_job.json")

        body = {}
        with open(file, 'r') as f:
            body = json.load(f)

        self.ml_es.put_job(job_id='test', body=body)

    def delete_ml_job(self):
        response = self.ml_es.get_jobs()
        if response["count"] == 0:
            return

        self.ml_es.delete_job(job_id='test')

    def create_ccr_stats(self):
        self.setup_ccr_remote()
        self.create_ccr_leader_index()
        self.create_ccr_follower_index()

    def setup_ccr_remote(self):
        file = os.path.join(self.beat_path, "module", "elasticsearch", "ccr",
                            "_meta", "test", "test_remote_settings.json")

        body = {}
        with open(file, 'r') as f:
            body = json.load(f)

        path = "/_cluster/settings"
        self.es.transport.perform_request('PUT', path, body=body)

    def create_ccr_leader_index(self):
        file = os.path.join(self.beat_path, "module", "elasticsearch", "ccr", "_meta", "test", "test_leader_index.json")

        body = {}
        with open(file, 'r') as f:
            body = json.load(f)

        path = "/pied_piper"
        self.es.transport.perform_request('PUT', path, body=body)

    def create_ccr_follower_index(self):
        file = os.path.join(self.beat_path, "module", "elasticsearch", "ccr",
                            "_meta", "test", "test_follower_index.json")

        body = {}
        with open(file, 'r') as f:
            body = json.load(f)

        path = "/rats/_ccr/follow"
        self.es.transport.perform_request('PUT', path, body=body)

    def ccr_unfollow_index(self):
        exists = self.es.indices.exists('rats')
        if not exists:
            return

        self.es.transport.perform_request('POST', '/rats/_ccr/pause_follow')
        self.es.indices.close('rats')
        self.es.transport.perform_request('POST', '/rats/_ccr/unfollow')

    def start_trial(self):
        # Check if trial is already enabled
        response = self.es.transport.perform_request('GET', self.license_url)
        if response["license"]["type"] == "trial":
            return

        # Enable xpack trial
        try:
            self.es.transport.perform_request('POST', self.license_url + "/start_trial?acknowledge=true")
        except:
            e = sys.exc_info()[0]
            print "Trial already enabled. Error: {}".format(e)

    def check_skip(self, metricset):
        if metricset != "ccr":
            return

        es_version = self.get_version()
        if es_version["major"] <= 6 and es_version["minor"] < 5:
            # Skip CCR metricset system test for Elasticsearch versions < 6.5.0 as CCR Stats
            # API endpoint is not available
            raise SkipTest("elasticsearch/ccr metricset system test only valid with Elasticsearch versions >= 6.5.0")

    def get_version(self):
        es_info = self.es.info()
        return semver.parse(es_info["version"]["number"])
