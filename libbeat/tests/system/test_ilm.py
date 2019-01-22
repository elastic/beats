from base import BaseTest
import os
from elasticsearch import Elasticsearch, TransportError
from nose.plugins.attrib import attr
import unittest
import shutil
import logging
import datetime

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class Test(BaseTest):

    def setUp(self):
        super(BaseTest, self).setUp()

        self.elasticsearch_url = self.get_elasticsearch_url()
        print("Using elasticsearch: {}".format(self.elasticsearch_url))
        self.es = Elasticsearch([self.elasticsearch_url])
        self.default_policy = "beatDefaultPolicy"
        self.default_pattern = "000001"
        logging.getLogger("urllib3").setLevel(logging.WARNING)
        logging.getLogger("elasticsearch").setLevel(logging.ERROR)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_enabled_false(self):
        """
        Test ilm false
        """

        rollover_alias = "mockbeat-rollover"
        template_name = "mockbeat"
        self.render_config_template("mockbeat",
                                    indices=[{"index": "mockbeat-index",
                                              "ilm.enabled": "false",
                                              "ilm.rollover_alias": rollover_alias,
                                              "template.name": template_name}],
                                    elasticsearch={"hosts": '["{}"]'.format(self.get_host())})

        self.clean(rollover_alias)

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains("PublishEvents: 1 events have been published"))
        proc.check_kill_and_wait()

        # Check if template is loaded with settings
        template = self.es.transport.perform_request('GET', '/_template/' + template_name)

        assert "lifecycle" not in template[template_name]["settings"]["index"]

        # Asserts that data is actually written to the index
        data = self.es.transport.perform_request('GET', '/mockbeat-index/_search')
        assert data["hits"]["total"] > 0

        # Check that policy is not stored
        policies = self.es.transport.perform_request('GET', '/_ilm/policy')
        assert self.default_policy not in policies

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_default(self):
        """
        Test ilm default settings
        """

        rollover_alias = "mockbeat-rollover"
        template_name = "mockbeat"
        self.render_config_template("mockbeat",
                                    elasticsearch={"hosts": '["{}"]'.format(self.get_host())})

        self.check_indices(rollover_alias, template_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_enabled_auto(self):
        """
        Test ilm auto
        """

        rollover_alias = "mockbeat-rollover"
        template_name = "mockbeat"
        self.render_config_template("mockbeat",
                                    indices=[{"index": "mockbeat-index",
                                              "ilm.enabled": "auto",
                                              "ilm.rollover_alias": rollover_alias,
                                              "template.name": template_name}],
                                    elasticsearch={"hosts": '["{}"]'.format(self.get_host())})

        self.check_indices(rollover_alias, template_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_enabled_true(self):
        """
        Test ilm true
        """

        rollover_alias = "mockbeat-rollover"
        template_name = "mockbeat"
        self.render_config_template("mockbeat",
                                    indices=[{"index": "mockbeat-index",
                                              "ilm.enabled": True,
                                              "ilm.rollover_alias": rollover_alias,
                                              "template.name": template_name}],
                                    elasticsearch={"hosts": '["{}"]'.format(self.get_host())})

        self.check_indices(rollover_alias, template_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_date_pattern(self):
        """
        Test ilm date pattern
        """

        rollover_alias = "mockbeat-rollover"
        template_name = "mockbeat"
        pattern = "'{now/d}'"
        index = rollover_alias + "-" + datetime.datetime.now().strftime("%Y.%m.%d")
        self.render_config_template("mockbeat",
                                    indices=[{"index": "mockbeat-index",
                                              "ilm.enabled": "auto",
                                              "ilm.rollover_alias": rollover_alias,
                                              "ilm.pattern": pattern,
                                              "template.name": template_name}],
                                    elasticsearch={"hosts": '["{}"]'.format(self.get_host())})

        self.check_indices(rollover_alias, template_name, pattern=pattern, index=index)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_policy(self):
        """
        Test ilm configured policy
        """

        rollover_alias = "mockbeat-rollover"
        template_name = "mockbeat"
        policy = "deleteAfter10Days"
        self.render_config_template("mockbeat",
                                    indices=[{"index": "mockbeat-index",
                                              "ilm.enabled": True,
                                              "ilm.rollover_alias": rollover_alias,
                                              "ilm.policy.name": policy,
                                              "template.name": template_name}],
                                    elasticsearch={"hosts": '["{}"]'.format(self.get_host())})

        self.check_indices(rollover_alias, template_name, policy=policy)

    def check_indices(self, rollover_alias, template_name, policy="", pattern="", index=""):
        if policy == "":
            policy = self.default_policy
        if pattern == "":
            pattern = self.default_pattern
        if index == "":
            index = rollover_alias + "-" + pattern

        self.clean(rollover_alias)

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains("PublishEvents: 1 events have been published"))
        proc.check_kill_and_wait()

        # Check if template is loaded with settings
        template = self.es.transport.perform_request('GET', '/_template/' + template_name)

        assert template[template_name]["settings"]["index"]["lifecycle"]["name"] == policy
        assert template[template_name]["settings"]["index"]["lifecycle"]["rollover_alias"] == rollover_alias

        # Make sure the correct index + alias was created
        alias = self.es.transport.perform_request('GET', '/_alias/' + rollover_alias)
        assert alias[index]["aliases"][rollover_alias]["is_write_index"] == True

        # Asserts that data is actually written to the ILM indices
        data = self.es.transport.perform_request('GET', '/' + index + '/_search')
        assert data["hits"]["total"] > 0

        # Check that policy is stored
        policies = self.es.transport.perform_request('GET', '/_ilm/policy')
        assert policy in policies

    def clean(self, alias_name):
        # Delete existing indices and aliases with it policy
        try:
            self.es.transport.perform_request('DELETE', "/" + alias_name + "*")
        except:
            pass

        # Delete any existing policy
        policies = self.es.transport.perform_request('GET', '/_ilm/policy')
        for p in policies:
            self.es.transport.perform_request('DELETE', "/_ilm/policy/" + p)

        # Delete templates
        try:
            self.es.transport.perform_request('DELETE', "/_template/mockbeat*")
        except:
            pass

        # Delete indices
        try:
            self.es.transport.perform_request('DELETE', "/mockbeat*")
        except:
            pass

    def get_host(self):
        return os.getenv('ES_HOST', 'localhost') + ':' + os.getenv('ES_PORT', '9200')
