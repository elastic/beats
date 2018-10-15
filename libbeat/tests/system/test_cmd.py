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
    def test_setup_flag(self):
        """
        Test --setup flag on run command
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

        proc = self.start_beat(
            extra_args=["--setup",
                        "--path.config", self.working_dir,
                        "-E", "setup.dashboards.file=" +
                        os.path.join(self.beat_path, "tests", "files", "testbeat-dashboards.zip"),
                        "-E", "setup.dashboards.beat=testbeat",
                        "-E", "setup.kibana.protocol=http",
                        "-E", "setup.kibana.host=" + self.get_kibana_host(),
                        "-E", "setup.kibana.port=" + self.get_kibana_port(),
                        "-E", "output.elasticsearch.hosts=['" + self.get_host() + "']"],
            config="libbeat.yml")

        self.wait_until(lambda: self.es.cat.templates(name='mockbeat-*', h='name') > 0)
        self.wait_until(lambda: self.log_contains("Kibana dashboards successfully loaded"))
        proc.check_kill_and_wait()

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_test_config(self):
        """
        Test test config command
        """
        self.render_config_template("mockbeat",
                                    os.path.join(self.working_dir, "libbeat.yml"))

        exit_code = self.run_beat(
            logging_args=[],
            extra_args=["test", "config"],
            config="libbeat.yml")

        assert exit_code == 0
        assert self.log_contains("Config OK")

    def test_test_bad_config(self):
        """
        Test test config command with bad config
        """
        exit_code = self.run_beat(
            logging_args=[],
            extra_args=["test", "config"],
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
                                    os.path.join(self.working_dir,
                                                 "mockbeat.yml"),
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

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_test_output(self):
        """
        Test test output works
        """
        self.render_config_template("mockbeat",
                                    os.path.join(self.working_dir,
                                                 "mockbeat.yml"),
                                    elasticsearch={"hosts": '["{}"]'.format(self.get_host())})
        exit_code = self.run_beat(
            extra_args=["test", "output"],
            config="mockbeat.yml")

        assert exit_code == 0
        assert self.log_contains('parse url... OK')
        assert self.log_contains('TLS... WARN secure connection disabled')
        assert self.log_contains('talk to server... OK')

    def test_test_wrong_output(self):
        """
        Test test wrong output works
        """
        self.render_config_template("mockbeat",
                                    os.path.join(self.working_dir,
                                                 "mockbeat.yml"),
                                    elasticsearch={"hosts": '["badhost:9200"]'})
        exit_code = self.run_beat(
            extra_args=["test", "output"],
            config="mockbeat.yml")

        assert exit_code == 1
        assert self.log_contains('parse url... OK')
        assert self.log_contains('dns lookup... ERROR')

    def get_host(self):
        return os.getenv('ES_HOST', 'localhost') + ':' + os.getenv('ES_PORT', '9200')

    def get_kibana_host(self):
        return os.getenv('KIBANA_HOST', 'localhost')

    def get_kibana_port(self):
        return os.getenv('KIBANA_PORT', '5601')
