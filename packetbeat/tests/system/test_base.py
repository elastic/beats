import os
import sys
import unittest
import pytest
import semver
import requests
import shutil
from beat import common_tests
from beat.beat import INTEGRATION_TESTS
from elasticsearch import Elasticsearch
from packetbeat import BaseTest


class Test(BaseTest, common_tests.TestExportsMixin):

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
