from base import BaseTest
import os
from elasticsearch import Elasticsearch, TransportError
from nose.plugins.attrib import attr
import unittest

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class Test(BaseTest):

    def test_index_modified(self):
        """
        Test that beat stops in case elasticsearch index is modified and pattern not
        """
        self.render_config_template(
            elasticsearch={"index": "test"},
        )

        exit_code = self.run_beat()

        assert exit_code == 1
        assert self.log_contains(
            "setup.template.name and setup.template.pattern have to be set if index name is modified") is True

    def test_index_not_modified(self):
        """
        Test that beat starts running if elasticsearch output is set
        """
        self.render_config_template(
            elasticsearch={"hosts": "localhost:9200"},
        )

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        proc.check_kill_and_wait()

    def test_index_modified_no_pattern(self):
        """
        Test that beat stops in case elasticsearch index is modified and pattern not
        """
        self.render_config_template(
            elasticsearch={"index": "test"},
            es_template_name="test",
        )

        exit_code = self.run_beat()

        assert exit_code == 1
        assert self.log_contains(
            "setup.template.name and setup.template.pattern have to be set if index name is modified") is True

    def test_index_modified_no_name(self):
        """
        Test that beat stops in case elasticsearch index is modified and name not
        """
        self.render_config_template(
            elasticsearch={"index": "test"},
            es_template_pattern="test",
        )

        exit_code = self.run_beat()

        assert exit_code == 1
        assert self.log_contains(
            "setup.template.name and setup.template.pattern have to be set if index name is modified") is True

    def test_index_with_pattern_name(self):
        """
        Test that beat starts running if elasticsearch output with modified index and pattern and name are set
        """
        self.render_config_template(
            elasticsearch={"hosts": "localhost:9200"},
            es_template_name="test",
            es_template_pattern="test-*",
        )

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        proc.check_kill_and_wait()

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_json_template(self):
        """
        Test loading of json based template
        """

        self.copy_files(["template.json"])

        path = os.path.join(self.working_dir, "template.json")

        print path
        self.render_config_template(
            elasticsearch={"hosts": self.get_host()},
            template_overwrite="true",
            template_json_enabled="true",
            template_json_path=path,
            template_json_name="bla",
        )

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains("Loading json template from file"))
        self.wait_until(lambda: self.log_contains("Elasticsearch template with name 'bla' loaded"))
        proc.check_kill_and_wait()

        es = Elasticsearch([self.get_elasticsearch_url()])
        result = es.transport.perform_request('GET', '/_template/bla')
        assert len(result) == 1

    def get_host(self):
        return os.getenv('ES_HOST', 'localhost') + ':' + os.getenv('ES_PORT', '9200')
