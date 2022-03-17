import os
import pytest
import re
import requests
import semver
import shutil
import sys
import unittest

from metricbeat import BaseTest

from beat.beat import INTEGRATION_TESTS
from beat import common_tests
from elasticsearch import Elasticsearch


class Test(BaseTest, common_tests.TestExportsMixin):

    COMPOSE_SERVICES = ['elasticsearch', 'kibana']

    @unittest.skipUnless(re.match("(?i)win|linux|darwin|freebsd|openbsd", sys.platform), "os")
    def test_start_stop(self):
        """
        Metricbeat starts and stops without error.
        """
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["cpu"],
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("start running"))
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        # Ensure all Beater stages are used.
        assert self.log_contains("Setup Beat: metricbeat")
        assert self.log_contains("metricbeat start running")
        assert self.log_contains("metricbeat stopped")

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    def test_index_management(self):
        """
        Test that the template can be loaded with `setup --index-management`
        """
        es = Elasticsearch([self.get_elasticsearch_url()])
        self.render_config_template(
            modules=[{
                "name": "apache",
                "metricsets": ["status"],
                "hosts": ["localhost"],
            }],
            elasticsearch={"host": self.get_elasticsearch_url()},
        )
        exit_code = self.run_beat(extra_args=["setup", "--index-management", "-E", "setup.template.overwrite=true"])

        assert exit_code == 0
        assert self.log_contains('Loaded index template')
        assert len(es.cat.templates(name='metricbeat-*', h='name')) > 0

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @pytest.mark.timeout(8*60, func_only=True)
    def test_dashboards(self):
        """
        Test that the dashboards can be loaded with `setup --dashboards`
        """
        if self.is_saved_object_api_available():
            raise unittest.SkipTest(
                "Kibana Saved Objects API is used since 7.15")

        shutil.copytree(self.kibana_dir(), os.path.join(self.working_dir, "kibana"))

        es = Elasticsearch([self.get_elasticsearch_url()])
        self.render_config_template(
            modules=[{
                "name": "apache",
                "metricsets": ["status"],
                "hosts": ["localhost"],
            }],
            elasticsearch={"host": self.get_elasticsearch_url()},
            kibana={"host": self.get_kibana_url()},
        )
        exit_code = self.run_beat(extra_args=["setup", "--dashboards"])

        assert exit_code == 0, 'Error output: ' + self.get_log()
        assert self.log_contains("Kibana dashboards successfully loaded.")

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    def test_migration(self):
        """
        Test that the template loads when migration is enabled
        """

        es = Elasticsearch([self.get_elasticsearch_url()])
        self.render_config_template(
            modules=[{
                "name": "apache",
                "metricsets": ["status"],
                "hosts": ["localhost"],
            }],
            elasticsearch={"host": self.get_elasticsearch_url()},
        )
        exit_code = self.run_beat(extra_args=["setup", "--index-management",
                                              "-E", "setup.template.overwrite=true", "-E", "migration.6_to_7.enabled=true"])

        assert exit_code == 0
        assert self.log_contains('Loaded index template')
        assert len(es.cat.templates(name='metricbeat-*', h='name')) > 0

    def get_elasticsearch_url(self):
        return "http://" + self.compose_host("elasticsearch")

    def get_kibana_url(self):
        """
        Returns kibana host URL
        """
        return "http://" + self.compose_host("kibana")

    def kibana_dir(self):
        return os.path.join(self.beat_path, "build", "kibana")

    def is_saved_object_api_available(self):
        kibana_semver = semver.VersionInfo.parse(self.get_version())
        return semver.VersionInfo.parse("7.14.0") <= kibana_semver

    def get_version(self):
        url = self.get_kibana_url() + "/api/status"

        r = requests.get(url)
        body = r.json()
        version = body["version"]["number"]

        return version
