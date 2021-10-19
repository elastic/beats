import os
import unittest
import pytest
import semver
import requests
import shutil
from filebeat import BaseTest
from elasticsearch import Elasticsearch
from beat.beat import INTEGRATION_TESTS
from beat import common_tests


class Test(BaseTest, common_tests.TestExportsMixin):

    def test_base(self):
        """
        Test if the basic fields exist.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
        )

        with open(self.working_dir + "/test.log", "w") as f:
            f.write("test message\n")

        filebeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=1))
        filebeat.check_kill_and_wait()

        output = self.read_output()[0]
        assert "@timestamp" in output
        assert "input.type" in output

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    def test_template(self):
        """
        Test that the template can be loaded with `setup --template`
        """
        es = Elasticsearch([self.get_elasticsearch_url()])
        self.render_config_template(
            elasticsearch={"host": self.get_elasticsearch_url()},
        )
        exit_code = self.run_beat(extra_args=["setup", "--template"])

        assert exit_code == 0
        assert self.log_contains('Loaded index template')
        assert len(es.cat.templates(name='filebeat-*', h='name')) > 0

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    def test_template_migration(self):
        """
        Test that the template can be loaded with `setup --template`
        """
        es = Elasticsearch([self.get_elasticsearch_url()])
        self.render_config_template(
            elasticsearch={"host": self.get_elasticsearch_url()},
        )
        exit_code = self.run_beat(extra_args=["setup", "--template",
                                              "-E", "setup.template.overwrite=true", "-E", "migration.6_to_7.enabled=true"])

        assert exit_code == 0
        assert self.log_contains('Loaded index template')
        assert len(es.cat.templates(name='filebeat-*', h='name')) > 0

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @pytest.mark.timeout(5*60, func_only=True)
    def test_dashboards(self):
        """
        Test that the dashboards can be loaded with `setup --dashboards`
        """
        if not self.is_saved_object_api_available():
            raise unittest.SkipTest(
                "Kibana Saved Objects API is used since 7.15")

        shutil.copytree(self.kibana_dir(), os.path.join(self.working_dir, "kibana"))

        es = Elasticsearch([self.get_elasticsearch_url()])
        self.render_config_template(
            elasticsearch={"host": self.get_elasticsearch_url()},
            kibana={"host": self.get_kibana_url()},
        )
        exit_code = self.run_beat(extra_args=["setup", "--dashboards"])

        assert exit_code == 0, 'Error output: ' + self.get_log()
        assert self.log_contains("Kibana dashboards successfully loaded.")

    def is_saved_object_api_available(self):
        kibana_semver = semver.VersionInfo.parse(self.get_version())
        return semver.VersionInfo.parse("7.14.0") <= kibana_semver

    def get_version(self):
        url = self.get_kibana_url() + "/api/status"

        r = requests.get(url)
        body = r.json()
        version = body["version"]["number"]

        return version

    def kibana_dir(self):
        return os.path.join(self.beat_path, "build", "kibana")
