import base64
import json
import metricbeat
import os
import semver
import sys
import unittest
from elasticsearch import Elasticsearch, NotFoundError
from parameterized import parameterized


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['elasticsearch']
    FIELDS = ["elasticsearch"]

    def setUp(self):
        super(Test, self).setUp()
        self.username = os.getenv("ES_USER", "")
        self.password = os.getenv("ES_PASS", "")
        self.es = Elasticsearch([self.get_elasticsearch_url()], basic_auth=(self.username, self.password))
        self.auth_value = base64.b64encode(f"{self.username}:{self.password}".encode()).decode()
        self.getHeaders = {"Authorization": f"Basic {self.auth_value}"}
        self.postHeaders = {"Authorization": f"Basic {self.auth_value}", "Content-Type": "application/json"}

        es_version = self.get_version()
        if es_version["major"] < 7:
            self.license_url = "/_xpack/license"
            self.ml_anomaly_detectors_url = "/_xpack/ml/anomaly_detectors"
        else:
            self.license_url = "/_license"
            self.ml_anomaly_detectors_url = "/_ml/anomaly_detectors"

        self.start_trial()

        try:
            self.es.indices.delete(index='test_index')
        except NotFoundError:
            pass

        self.es.indices.create(index='test_index')

    def tearDown(self):
        self.ccr_unfollow_index()
        try:
            self.es.indices.delete(index='test_index,pied_piper,rats')
        except NotFoundError:
            pass
        self.delete_ml_job()
        self.delete_enrich_ingest_pipeline()
        self.delete_enrich_policy()
        try:
            self.es.indices.delete(index='users,my_index')
        except NotFoundError:
            pass
        super(Test, self).tearDown()

    @parameterized.expand([
        "ccr",
        "enrich",
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
        if metricset == "enrich":
            self.create_enrich_stats()

        self.check_metricset("elasticsearch", metricset, [self.get_elasticsearch_url()], self.FIELDS +
                             ["service"], extras={"index_recovery.active_only": "false"})

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_xpack(self):
        """
        elasticsearch-xpack module tests
        """

        self.create_ml_job()
        if self.is_ccr_available():
            self.create_ccr_stats()
        if self.is_enrich_available():
            self.create_enrich_stats()

        self.render_config_template(modules=[{
            "name": "elasticsearch",
            "metricsets": [
                "ccr",
                "cluster_stats",
                "enrich",
                "index",
                "index_recovery",
                "index_summary",
                "ml_job",
                "shard"
            ],
            "hosts": self.get_elasticsearch_url(),
            "period": "1s",
            "extras": {
                "xpack.enabled": "true"
            }
        }])

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_xpack_cluster_stats(self):
        """
        elasticsearch-xpack module test for type:cluster_stats
        """

        self.render_config_template(modules=[{
            "name": "elasticsearch",
            "metricsets": [
                "ccr",
                "cluster_stats",
                "enrich",
                "index",
                "index_recovery",
                "index_summary",
                "ml_job",
                "node_stats",
                "shard"
            ],
            "hosts": self.get_elasticsearch_url(),
            "period": "1s",
            "extras": {
                "xpack.enabled": "true"
            }
        }])
        proc = self.start_beat()
        self.wait_log_contains('\\"dataset\\": \\"elasticsearch.cluster.stats\\"')

        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        docs = self.read_output_json()
        for doc in docs:
            t = doc["metricset"]["name"]
            if t != "cluster_stats":
                continue
            license = doc["elasticsearch"]["cluster"]["stats"]["license"]
            issue_date = license["issue_date_in_millis"]
            self.assertIsNot(type(issue_date), float)

            self.assertNotIn("expiry_date_in_millis", license)

    def create_ml_job(self):
        # Check if an ml job already exists
        response = self.es.ml.get_jobs()
        if response.body["count"] > 0:
            return

        file = os.path.join(self.beat_path, "module", "elasticsearch", "ml_job", "_meta", "test", "test_job.json")

        body = {}
        with open(file, 'r') as f:
            body = json.load(f)

        self.es.ml.put_job(job_id='test', data_description=body["data_description"],
                           analysis_config=body["analysis_config"], description=body["description"])

    def delete_ml_job(self):
        response = self.es.ml.get_jobs()
        if response.body["count"] == 0:
            return

        self.es.ml.delete_job(job_id='test')

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
        self.es.transport.perform_request('PUT', path, body=body, headers=self.postHeaders)

    def create_ccr_leader_index(self):
        file = os.path.join(self.beat_path, "module", "elasticsearch", "ccr", "_meta", "test", "test_leader_index.json")

        body = {}
        with open(file, 'r') as f:
            body = json.load(f)

        path = "/pied_piper"
        self.es.transport.perform_request('PUT', path, body=body, headers=self.postHeaders)

    def create_ccr_follower_index(self):
        file = os.path.join(self.beat_path, "module", "elasticsearch", "ccr",
                            "_meta", "test", "test_follower_index.json")

        body = {}
        with open(file, 'r') as f:
            body = json.load(f)

        path = "/rats/_ccr/follow"
        self.es.transport.perform_request('PUT', path, body=body, headers=self.postHeaders)

    def ccr_unfollow_index(self):
        exists = self.es.indices.exists(index='rats')
        if not exists:
            return

        self.es.transport.perform_request('POST', '/rats/_ccr/pause_follow', headers=self.postHeaders)
        self.es.transport.perform_request('POST', '/rats/_ccr/unfollow', headers=self.postHeaders)

    def create_enrich_stats(self):
        self.create_enrich_source_index()
        self.create_enrich_policy()
        self.execute_enrich_policy()
        self.create_enrich_ingest_pipeline()
        self.ingest_and_enrich_doc()

    def create_enrich_source_index(self):
        file = os.path.join(self.beat_path, 'module', 'elasticsearch', 'enrich',
                            '_meta', 'test', 'source_doc.json')

        source_doc = {}
        with open(file, 'r') as f:
            source_doc = json.load(f)

        self.es.index(index='users', id='1', document=source_doc, refresh='wait_for')

    def create_enrich_policy(self):
        file = os.path.join(self.beat_path, 'module', 'elasticsearch', 'enrich',
                            '_meta', 'test', 'policy.json')

        policy = {}
        with open(file, 'r') as f:
            policy = json.load(f)

        policy_url = '/_enrich/policy/users-policy'
        self.es.transport.perform_request(method='PUT', target=policy_url, body=policy, headers=self.postHeaders)

    def execute_enrich_policy(self):
        execute_url = '/_enrich/policy/users-policy/_execute'
        self.es.transport.perform_request('POST', execute_url, headers=self.postHeaders)

    def create_enrich_ingest_pipeline(self):
        file = os.path.join(self.beat_path, 'module', 'elasticsearch', 'enrich',
                            '_meta', 'test', 'ingest_pipeline.json')

        pipeline = {}
        with open(file, 'r') as f:
            pipeline = json.load(f)

        self.es.ingest.put_pipeline(
            id='user_lookup', description=pipeline['description'], processors=pipeline['processors'])

    def ingest_and_enrich_doc(self):
        file = os.path.join(self.beat_path, 'module', 'elasticsearch', 'enrich',
                            '_meta', 'test', 'target_doc.json')

        target_doc = {}
        with open(file, 'r') as f:
            target_doc = json.load(f)

        self.es.index(index='my_index', id='my_id', document=target_doc, pipeline='user_lookup')

    def delete_enrich_policy(self):
        exists = self.es.indices.exists(index='my_index')
        if not exists:
            return

        self.es.transport.perform_request('DELETE', '/_enrich/policy/users-policy', headers=self.getHeaders)

    def delete_enrich_ingest_pipeline(self):
        exists = self.es.indices.exists(index='my_index')
        if not exists:
            return

        self.es.ingest.delete_pipeline(id='user_lookup')

    def start_trial(self):
        # Check if trial is already enabled
        response = self.es.transport.perform_request('GET', self.license_url, headers=self.getHeaders)
        if response.body["license"]["type"] == "trial":
            return

        # Enable xpack trial
        try:
            resp = self.es.transport.perform_request('POST', self.license_url +
                                                     "/start_trial?acknowledge=true", headers=self.postHeaders)
        except BaseException:
            e = sys.exc_info()[0]
            print("Trial already enabled. Error: {}".format(e))

    def start_basic(self):
        # Check if basic license is already enabled
        response = self.es.transport.perform_request('GET', self.license_url, headers=self.getHeaders)
        if response.body["license"]["type"] == "basic":
            return

        try:
            self.es.transport.perform_request('POST', self.license_url +
                                              "/start_basic?acknowledge=true", headers=self.postHeaders)
        except BaseException:
            e = sys.exc_info()[0]
            print("Basic license already enabled. Error: {}".format(e))

    def check_skip(self, metricset):
        if metricset == 'ccr' and not self.is_ccr_available():
            raise unittest.SkipTest(
                "elasticsearch/ccr metricset system test only valid with Elasticsearch versions >= 6.5.0")

        if metricset == 'enrich' and not self.is_enrich_available():
            raise unittest.SkipTest(
                "elasticsearch/enrich metricset system test only valid with Elasticsearch versions >= 7.5.0")

    def is_ccr_available(self):
        es_version = self.get_version()
        major = es_version["major"]
        minor = es_version["minor"]

        return major > 6 or (major == 6 and minor >= 5)

    def is_enrich_available(self):
        es_version = self.get_version()
        major = es_version["major"]
        minor = es_version["minor"]

        return major > 7 or (major == 7 and minor >= 5)

    def get_version(self):
        es_info = self.es.info()
        return semver.parse(es_info["version"]["number"])
