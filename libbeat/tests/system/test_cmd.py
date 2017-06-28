from base import BaseTest
from nose.plugins.attrib import attr
from elasticsearch import Elasticsearch, TransportError

import logging
import os
import shutil
import unittest


INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class TestCommands(BaseTest):
    """
    Test beat subcommands
    """

    def setUp(self):
        super(BaseTest, self).setUp()

        self.elasticsearch_url = self.get_elasticsearch_url()
        print("Using elasticsearch: {}".format(self.elasticsearch_url))
        self.es = Elasticsearch([self.elasticsearch_url])
        logging.getLogger("urllib3").setLevel(logging.WARNING)
        logging.getLogger("elasticsearch").setLevel(logging.ERROR)

    def test_version(self):
        """
        Test version command
        """
        exit_code = self.run_beat(
            extra_args=["version"], logging_args=["-v", "-d", "*"])
        assert exit_code == 0

        assert self.log_contains("mockbeat") is True
        assert self.log_contains("version") is True
        assert self.log_contains("9.9.9") is True

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_setup_template(self):
        """
        Test setup -template command
        """
        # Delete any existing template
        try:
            self.es.indices.delete_template('mockbeat-*')
        except:
            pass

        assert len(self.es.cat.templates(name='mockbeat-*', h='name')) == 0

        shutil.copy(self.beat_path + "/_meta/config.yml",
                    os.path.join(self.working_dir, "libbeat.yml"))
        shutil.copy(self.beat_path + "/fields.yml",
                    os.path.join(self.working_dir, "fields.yml"))

        exit_code = self.run_beat(
            logging_args=["-v", "-d", "*"],
            extra_args=["setup",
                        "-template",
                        "-path.config", self.working_dir,
                        "-E", "output.elasticsearch.hosts=['" + self.get_host() + "']"],
            config="libbeat.yml")

        assert exit_code == 0
        assert len(self.es.cat.templates(name='mockbeat-*', h='name')) > 0

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_configtest(self):
        """
        Test configtest command
        """
        self.render_config_template("mockbeat",
                                    os.path.join(self.working_dir, "libbeat.yml"))

        exit_code = self.run_beat(
            logging_args=[],
            extra_args=["configtest"],
            config="libbeat.yml")

        assert exit_code == 0
        assert self.log_contains("Config OK")

    def test_configtest_bad_config(self):
        """
        Test configtest command with bad config
        """
        exit_code = self.run_beat(
            logging_args=[],
            extra_args=["configtest"],
            config="libbeat-missing.yml")

        assert exit_code == 1
        assert self.log_contains("Config OK") is False

    def test_export_config(self):
        """
        Test export config works
        """
        self.render_config_template("mockbeat",
                                    os.path.join(self.working_dir,
                                                 "libbeat.yml"),
                                    metrics_period=1234)

        exit_code = self.run_beat(
            logging_args=[],
            extra_args=["export", "config"],
            config="libbeat.yml")

        assert exit_code == 0
        assert self.log_contains("filename: mockbeat")
        assert self.log_contains("period: 1234")

    def test_export_template(self):
        """
        Test export template works
        """
        self.render_config_template("mockbeat",
                                    os.path.join(self.working_dir, "mockbeat.yml"),
                                    fields=os.path.join(self.working_dir, "fields.yml"))
        shutil.copy(self.beat_path + "/fields.yml",
                    os.path.join(self.working_dir, "fields.yml"))
        exit_code = self.run_beat(
            logging_args=[],
            extra_args=["export", "template"],
            config="mockbeat.yml")

        assert exit_code == 0
        assert self.log_contains('"mockbeat-9.9.9-*"')
        assert self.log_contains('"codec": "best_compression"')

    def get_host(self):
        return os.getenv('ES_HOST', 'localhost') + ':' + os.getenv('ES_PORT', '9200')
