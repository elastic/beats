import json
import os
import pytest
import requests
import semver
import shutil
import unittest
import yaml

from elasticsearch import Elasticsearch
from beat.beat import INTEGRATION_TESTS

# Fail if the exported index pattern is larger than 10MiB
# This is to avoid problems with Kibana when the payload
# of the request to install the index pattern exceeds the
# default limit.
index_pattern_size_limit = 10 * 1024 * 1024


class TestExportsMixin:

    def run_export_cmd(self, cmd, extra=[]):
        """
        Runs the given export command and returns the output as a string.
        Raises an exception if the command fails.
        :param cmd: the export command
        :param extra: Extra arguments (optional)
        :return: The output as a string.
        """
        self.render_config_template()

        args = ["export", cmd]
        if len(extra) != 0:
            args += extra
        exit_code = self.run_beat(extra_args=args, logging_args=[])
        output = self.get_log()
        if exit_code != 0:
            raise Exception("export command returned with an error: {}".format(output))
        trailer = "\nPASS\n"
        pos = output.rfind(trailer)
        if pos == -1:
            raise Exception("didn't return expected trailer:{} got:{}".format(
                trailer.__repr__(),
                output[-100:].__repr__()))
        return output[:pos]

    def test_export_ilm_policy(self):
        """
        Test that the ilm-policy can be exported with `export ilm-policy`
        """
        output = self.run_export_cmd("ilm-policy")
        js = json.loads(output)
        assert "policy" in js

    def test_export_template(self):
        """
        Test that the template can be exported with `export template`
        """
        output = self.run_export_cmd("template")
        js = json.loads(output)
        assert "index_patterns" in js
        assert "template" in js
        assert "priority" in js
        assert "order" not in js
        assert "mappings" in js["template"]
        assert "settings" in js["template"]

    def test_export_index_pattern(self):
        """
        Test that the index-pattern can be exported with `export index-pattern`
        """
        output = self.run_export_cmd("index-pattern")
        js = json.loads(output)
        assert "attributes" in js
        assert "index-pattern" == js["type"]
        size = len(output.encode('utf-8'))
        assert size < index_pattern_size_limit, "Kibana index pattern must be less than 10MiB " \
            "to keep the Beat setup request size below " \
            "Kibana's server.maxPayloadBytes."

    def test_export_index_pattern_migration(self):
        """
        Test that the index-pattern can be exported with `export index-pattern` (migration enabled)
        """
        output = self.run_export_cmd("index-pattern", extra=['-E', 'migration.6_to_7.enabled=true'])
        js = json.loads(output)
        assert "attributes" in js
        assert "index-pattern" == js["type"]
        size = len(output.encode('utf-8'))
        assert size < index_pattern_size_limit, "Kibana index pattern must be less than 10MiB " \
            "to keep the Beat setup request size below " \
            "Kibana's server.maxPayloadBytes."

    def test_export_config(self):
        """
        Test that the config can be exported with `export config`
        """
        output = self.run_export_cmd("config")
        yml = yaml.load(output, Loader=yaml.FullLoader)
        assert isinstance(yml, dict)


class TestDashboardMixin:

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
