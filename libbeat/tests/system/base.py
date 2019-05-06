import os
from beat.beat import TestCase
from elasticsearch import Elasticsearch, NotFoundError
from nose.tools import raises
import logging
import datetime


class BaseTest(TestCase):

    @classmethod
    def setUpClass(self):
        self.beat_name = "mockbeat"
        self.beat_path = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../"))
        self.test_binary = self.beat_path + "/libbeat.test"
        self.beats = [
            "filebeat",
            "heartbeat",
            "metricbeat",
            "packetbeat",
            "winlogbeat"
        ]
        super(BaseTest, self).setUpClass()

    def clean(self):

        # Delete existing indices and aliases
        try:
            self.es.transport.perform_request('DELETE', "/*")
        except:
            pass

        # Delete templates
        try:
            self.es.transport.perform_request('DELETE', "/_template/*")
        except:
            pass

        # Delete any existing policy
        policies = self.es.transport.perform_request('GET', "/_ilm/policy")
        for policy, _ in policies.items():
            try:
                self.es.transport.perform_request('DELETE', "/_ilm/policy/" + policy)
            except:
                pass


class WithESTest(BaseTest):
    """
    Basis for integration test with Elasticsearch
    """

    def setUp(self):
        super(WithESTest, self).setUp()

        self.elasticsearch_url = self.get_elasticsearch_url()
        print("Using elasticsearch: {}".format(self.elasticsearch_url))
        self.es = Elasticsearch([self.elasticsearch_url])
        logging.getLogger("urllib3").setLevel(logging.WARNING)
        logging.getLogger("elasticsearch").setLevel(logging.ERROR)


class IndexAssertions(WithESTest):

    @raises(NotFoundError)
    def assert_index_template_not_loaded(self, template):
        self.es.transport.perform_request('GET', '/_template/' + template)

    def assert_index_template_loaded(self, template):
        resp = self.es.transport.perform_request('GET', '/_template/' + template)
        assert template in resp
        assert "lifecycle" not in resp[template]["settings"]["index"]

    def assert_ilm_template_loaded(self, template, policy, alias):
        resp = self.es.transport.perform_request('GET', '/_template/' + template)
        assert resp[template]["settings"]["index"]["lifecycle"]["name"] == policy
        assert resp[template]["settings"]["index"]["lifecycle"]["rollover_alias"] == alias

    def assert_index_template_index_pattern(self, template, index_pattern):
        resp = self.es.transport.perform_request('GET', '/_template/' + template)
        assert template in resp
        assert resp[template]["index_patterns"] == index_pattern

    def assert_alias_not_created(self, alias):
        resp = self.es.transport.perform_request('GET', '/_alias')
        for name, entry in resp.items():
            if alias not in name:
                continue
            assert entry["aliases"] == {}, entry["aliases"]

    def assert_alias_created(self, alias, pattern=None):
        if pattern is None:
            pattern = self.default_pattern()
        name = alias + "-" + pattern
        resp = self.es.transport.perform_request('GET', '/_alias/' + alias)
        assert name in resp
        assert resp[name]["aliases"][alias]["is_write_index"] == True

    @raises(NotFoundError)
    def assert_policy_not_created(self, policy):
        self.es.transport.perform_request('GET', '/_ilm/policy/' + policy)

    def assert_policy_created(self, policy):
        resp = self.es.transport.perform_request('GET', '/_ilm/policy/' + policy)
        assert policy in resp
        assert resp[policy]["policy"]["phases"]["hot"]["actions"]["rollover"]["max_size"] == "50gb"
        assert resp[policy]["policy"]["phases"]["hot"]["actions"]["rollover"]["max_age"] == "30d"

    def assert_docs_written_to_alias(self, alias, pattern=None):
        if pattern is None:
            pattern = self.default_pattern()
        name = alias + "-" + pattern
        self.wait_until(lambda: self.es.transport.perform_request(
            'GET', '/' + name + '/_search')["hits"]["total"] > 0)

        data = self.es.transport.perform_request('GET', '/' + name + '/_search')
        assert data["hits"]["total"] > 0

    def assert_log_contains_policy(self, policy):
        assert self.log_contains('ILM policy successfully loaded.')
        assert self.log_contains(policy)
        assert self.log_contains('"max_age": "30d"')
        assert self.log_contains('"max_size": "50gb"')

    def assert_log_contains_write_alias(self):
        assert self.log_contains('Write alias successfully generated.')

    def assert_log_contains_template(self, template, index_pattern):
        assert self.log_contains('Loaded index template')
        assert self.log_contains(template)
        assert self.log_contains(index_pattern)

    def default_pattern(self):
        d = datetime.datetime.now().strftime("%Y.%m.%d")
        return d + "-000001"
