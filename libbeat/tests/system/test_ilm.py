from base import BaseTest
import os
from elasticsearch import Elasticsearch, TransportError
from nose.plugins.attrib import attr
import unittest
import shutil
import logging

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class Test(BaseTest):

    def setUp(self):
        super(BaseTest, self).setUp()

        self.elasticsearch_url = self.get_elasticsearch_url()
        print("Using elasticsearch: {}".format(self.elasticsearch_url))
        self.es = Elasticsearch([self.elasticsearch_url])
        logging.getLogger("urllib3").setLevel(logging.WARNING)
        logging.getLogger("elasticsearch").setLevel(logging.ERROR)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_ilm_enabled(self):
        """
        Test ilm enabled
        """

        self.render_config_template(
            elasticsearch={
                "hosts": self.get_host(),
                "ilm.enabled": True,
            },
        )

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains("Overwriting setup.template for ILM"))
        proc.check_kill_and_wait()

        es = Elasticsearch([self.get_elasticsearch_url()])

        # Check if template is loaded with settings
        template = es.transport.perform_request('GET', '/_template/mockbeat-9.9.9')
        # TODO: check content
        print template

        # Make sure the correct index + alias was created
        alias = es.transport.perform_request('GET', '/_alias/mockbeat-9.9.9')
        assert "mockbeat-9.9.9-000001" in alias

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_setup_ilm_policy(self):
        """
        Test ilm policy setup
        """

        policy_name = "beats-default-policy"

        # Delete any existing policy
        try:
            self.es.transport.perform_request('DELETE', "/_ilm/policy/" + policy_name)
        except:
            pass

        shutil.copy(self.beat_path + "/_meta/config.yml",
                    os.path.join(self.working_dir, "libbeat.yml"))
        shutil.copy(self.beat_path + "/fields.yml",
                    os.path.join(self.working_dir, "fields.yml"))

        exit_code = self.run_beat(
            logging_args=["-v", "-d", "*"],
            extra_args=["setup",
                        "--ilm-policy",
                        "-path.config", self.working_dir,
                        "-E", "output.elasticsearch.hosts=['" + self.get_host() + "']"],
            config="libbeat.yml")

        assert exit_code == 0

        policy = self.es.transport.perform_request('GET', "/_ilm/policy/" + policy_name)
        assert policy_name in policy

    def get_host(self):
        return os.getenv('ES_HOST', 'localhost') + ':' + os.getenv('ES_PORT', '9200')
