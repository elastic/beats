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
        self.alias_name = "mockbeat-9.9.9"
        self.policy_name = "beats-default-policy"
        logging.getLogger("urllib3").setLevel(logging.WARNING)
        logging.getLogger("elasticsearch").setLevel(logging.ERROR)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_enabled(self):
        """
        Test ilm enabled
        """

        self.render_config_template(
            elasticsearch={
                "hosts": self.get_elasticsearch_url(),
                "ilm.enabled": True,
            },
        )

        self.clean()

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains("Set setup.template.name"))
        self.wait_until(lambda: self.log_contains("PublishEvents: 1 events have been published"))
        proc.check_kill_and_wait()

        # Check if template is loaded with settings
        template = self.es.transport.perform_request('GET', '/_template/' + self.alias_name)

        assert template[self.alias_name]["settings"]["index"]["lifecycle"]["name"] == "beats-default-policy"
        assert template[self.alias_name]["settings"]["index"]["lifecycle"]["rollover_alias"] == self.alias_name

        # Make sure the correct index + alias was created
        alias = self.es.transport.perform_request('GET', '/_alias/' + self.alias_name)
        d = datetime.datetime.now()
        now = d.strftime("%Y.%m.%d")
        index_name = self.alias_name + "-" + now + "-000001"
        assert index_name in alias
        assert alias[index_name]["aliases"][self.alias_name]["is_write_index"] == True

        # Asserts that data is actually written to the ILM indices
        self.wait_until(lambda: self.es.transport.perform_request(
            'GET', '/' + index_name + '/_search')["hits"]["total"] > 0)

        data = self.es.transport.perform_request('GET', '/' + index_name + '/_search')
        assert data["hits"]["total"] > 0

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_rollover_alias(self):
        """
        Test ilm rollover alias setting
        """

        alias_name = "foo"
        self.render_config_template(
            elasticsearch={
                "hosts": self.get_elasticsearch_url(),
                "ilm.enabled": True,
                "ilm.pattern": "1",
                "ilm.rollover_alias": alias_name
            },
        )

        self.clean(alias_name=alias_name)

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains("Set setup.template.name"))
        self.wait_until(lambda: self.log_contains("PublishEvents: 1 events have been published"))
        proc.check_kill_and_wait()

        # Make sure the correct index + alias was created
        print '/_alias/' + alias_name
        logfile = self.beat_name + ".log"
        with open(os.path.join(self.working_dir, logfile), "r") as f:
            print f.read()

        alias = self.es.transport.perform_request('GET', '/_alias/' + alias_name)
        index_name = alias_name + "-1"
        assert index_name in alias

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_pattern(self):
        """
        Test ilm pattern setting
        """

        self.render_config_template(
            elasticsearch={
                "hosts": self.get_elasticsearch_url(),
                "ilm.enabled": True,
                "ilm.pattern": "1"
            },
        )

        self.clean()

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains("Set setup.template.name"))
        self.wait_until(lambda: self.log_contains("PublishEvents: 1 events have been published"))
        proc.check_kill_and_wait()

        # Make sure the correct index + alias was created
        print '/_alias/' + self.alias_name
        logfile = self.beat_name + ".log"
        with open(os.path.join(self.working_dir, logfile), "r") as f:
            print f.read()

        alias = self.es.transport.perform_request('GET', '/_alias/' + self.alias_name)
        index_name = self.alias_name + "-1"
        assert index_name in alias

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_pattern_date(self):
        """
        Test ilm pattern with date inside
        """

        self.render_config_template(
            elasticsearch={
                "hosts": self.get_elasticsearch_url(),
                "ilm.enabled": True,
                "ilm.pattern": "'{now/d}'"
            },
        )

        self.clean()

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains("Set setup.template.name"))
        self.wait_until(lambda: self.log_contains("PublishEvents: 1 events have been published"))
        proc.check_kill_and_wait()

        # Make sure the correct index + alias was created
        print '/_alias/' + self.alias_name
        logfile = self.beat_name + ".log"
        with open(os.path.join(self.working_dir, logfile), "r") as f:
            print f.read()

        # Make sure the correct index + alias was created
        alias = self.es.transport.perform_request('GET', '/_alias/' + self.alias_name)
        d = datetime.datetime.now()
        now = d.strftime("%Y.%m.%d")
        index_name = self.alias_name + "-" + now
        assert index_name in alias
        assert alias[index_name]["aliases"][self.alias_name]["is_write_index"] == True

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_setup_ilm_policy(self):
        """
        Test ilm policy setup
        """

        self.clean()

        shutil.copy(self.beat_path + "/_meta/config.yml",
                    os.path.join(self.working_dir, "libbeat.yml"))
        shutil.copy(self.beat_path + "/fields.yml",
                    os.path.join(self.working_dir, "fields.yml"))

        exit_code = self.run_beat(
            logging_args=["-v", "-d", "*"],
            extra_args=["setup",
                        "--ilm-policy",
                        "-path.config", self.working_dir,
                        "-E", "output.elasticsearch.hosts=['" + self.get_elasticsearch_url() + "']"],
            config="libbeat.yml")

        assert exit_code == 0

        policy = self.es.transport.perform_request('GET', "/_ilm/policy/" + self.policy_name)
        assert self.policy_name in policy

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_export_ilm_policy(self):
        """
        Test ilm policy export
        """

        self.clean()

        shutil.copy(self.beat_path + "/_meta/config.yml",
                    os.path.join(self.working_dir, "libbeat.yml"))
        shutil.copy(self.beat_path + "/fields.yml",
                    os.path.join(self.working_dir, "fields.yml"))

        exit_code = self.run_beat(
            logging_args=["-v", "-d", "*"],
            extra_args=["export",
                        "ilm-policy",
                        ],
            config="libbeat.yml")

        assert exit_code == 0

        assert self.log_contains('"max_age": "30d"')
        assert self.log_contains('"max_size": "50gb"')

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
            self.es.transport.perform_request('DELETE', "/_ilm/policy/" + self.policy_name)
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
