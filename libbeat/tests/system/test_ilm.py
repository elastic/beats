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
        self.alias_name = "mockbeat-123"
        self.index = self.alias_name
        self.rollover_policy = "deleteAfter10Days"
        logging.getLogger("urllib3").setLevel(logging.WARNING)
        logging.getLogger("elasticsearch").setLevel(logging.ERROR)

    # @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_ilm_rollover_policy_configured(self):
        """
        Test ilm enabled
        """

        self.render_config_template(
            elasticsearch={
                "hosts": self.get_elasticsearch_url(),
                "index": self.alias_name,
                "rollover_policy": self.rollover_policy,
            },
            es_template_name=self.alias_name,
            es_template_pattern="{}*".format(self.alias_name)
        )

        self.clean()

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains("PublishEvents: 1 events have been published"))
        proc.check_kill_and_wait()

        # Check if template is loaded with settings
        template_name = "{}-ilm".format(self.alias_name)
        template = self.es.transport.perform_request('GET', '/_template/' + template_name)
        assert template[template_name]["settings"]["index"]["lifecycle"]["name"] == self.rollover_policy
        assert template[template_name]["settings"]["index"]["lifecycle"]["rollover_alias"] == self.alias_name

        # Make sure the correct index + alias was created
        alias = self.es.transport.perform_request('GET', '/_alias/' + self.alias_name)
        index_name = self.alias_name + "-000001"
        assert index_name in alias
        assert alias[index_name]["aliases"][self.alias_name]["is_write_index"] == True

        # Asserts that data is actually written to the ILM indices
        self.wait_until(lambda: self.es.transport.perform_request(
            'GET', '/' + index_name + '/_search')["hits"]["total"] > 0)

        data = self.es.transport.perform_request('GET', '/' + index_name + '/_search')
        assert data["hits"]["total"] > 0

    # @unittest.skipUnless(INTEGRATION_TESTS, "integration test")

    @attr('integration')
    def test_setup_configured_ilm_policy(self):
        """
        Test ilm policy setup
        """
        self.render_config_template(
            elasticsearch={
                "hosts": self.get_elasticsearch_url(),
                "index": self.alias_name,
                "rollover_policy": self.rollover_policy,
            },
            es_template_name=self.alias_name,
            es_template_pattern="{}*".format(self.alias_name)
        )
        self.clean()

        exit_code = self.run_beat(
            logging_args=["-v", "-d", "*"],
            extra_args=["setup",
                        "--ilm-policy",
                        "-path.config", self.working_dir,
                        "-E", "output.elasticsearch.hosts=['" + self.get_elasticsearch_url() + "']"])

        assert exit_code == 0

        policy = self.es.transport.perform_request('GET', "/_ilm/policy/" + self.rollover_policy)
        assert self.rollover_policy in policy

    @attr('integration')
    def test_export_configured_ilm_policy(self):
        """
        Test ilm policy export
        """
        self.render_config_template(
            elasticsearch={
                "hosts": self.get_elasticsearch_url(),
                "index": self.alias_name,
                "rollover_policy": self.rollover_policy,
            },
            es_template_name=self.alias_name,
            es_template_pattern="{}*".format(self.alias_name)
        )
        self.clean()

        exit_code = self.run_beat(
            logging_args=["-v", "-d", "*"],
            extra_args=["export",
                        "ilm-policy",
                        "-path.config", self.working_dir])

        assert exit_code == 0
        assert self.log_contains('"max_age":"1d"')
        assert self.log_contains('"max_size":"50gb"')

    def clean(self, alias_name=""):

        if alias_name == "":
            alias_name = self.alias_name

        # Delete existing indices and aliases with it policy
        try:
            self.es.transport.perform_request('DELETE', "/" + alias_name + "*")
        except:
            pass

        # Delete any existing policy
        try:
            self.es.transport.perform_request('DELETE', "/_ilm/policy/" + self.rollover_policy)
        except:
            pass

        # Delete templates
        try:
            self.es.transport.perform_request('DELETE', "/_template/mockbeat*")
        except:
            pass

        # Delete indices
        try:
            self.es.transport.perform_request('DELETE', "/foo*,mockbeat*")
        except:
            pass
