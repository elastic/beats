import os
import os.path
import pytest
import json
import re
import requests
import semver
import shutil
import subprocess
import unittest

from base import BaseTest

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class Test(BaseTest):

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_load_without_dashboard(self):
        """
        Test loading without dashboards
        """
        self.render_config_template()
        beat = self.start_beat(
            logging_args=["-e", "-d", "*"],
            extra_args=["setup",
                        "--dashboards",
                        "-E", "setup.dashboards.file=" +
                        os.path.join(self.beat_path, "tests", "files", "testbeat-no-dashboards.zip"),
                        "-E", "setup.dashboards.beat=testbeat",
                        "-E", "setup.kibana.protocol=http",
                        "-E", "setup.kibana.host=" + self.get_kibana_host(),
                        "-E", "setup.kibana.port=" + self.get_kibana_port(),
                        "-E", "setup.kibana.username=beats",
                        "-E", "setup.kibana.password=testing",
                        "-E", "output.elasticsearch.hosts=['" + self.get_host() + "']",
                        "-E", "output.elasticsearch.username=admin",
                        "-E", "output.elasticsearch.password=testing",
                        "-E", "output.file.enabled=false"]
        )

        beat.check_wait(exit_code=0)

        assert self.log_contains("Skipping loading dashboards")

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_load_dashboard(self):
        """
        Test loading dashboards
        """
        self.render_config_template()
        beat = self.start_beat(
            logging_args=["-e", "-d", "*"],
            extra_args=["setup",
                        "--dashboards",
                        "-E", "setup.dashboards.file=" +
                        os.path.join(self.beat_path, "tests", "files", "testbeat-dashboards.zip"),
                        "-E", "setup.dashboards.beat=testbeat",
                        "-E", "setup.kibana.protocol=http",
                        "-E", "setup.kibana.host=" + self.get_kibana_host(),
                        "-E", "setup.kibana.port=" + self.get_kibana_port(),
                        "-E", "setup.kibana.username=beats",
                        "-E", "setup.kibana.password=testing",
                        "-E", "output.elasticsearch.hosts=['" + self.get_host() + "']",
                        "-E", "output.elasticsearch.username=admin",
                        "-E", "output.elasticsearch.password=testing",
                        "-E", "output.file.enabled=false"]
        )
        beat.check_wait(exit_code=0)

        assert self.log_contains("Kibana dashboards successfully loaded") is True

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_load_only_index_patterns(self):
        """
        Test loading dashboards
        """
        self.render_config_template()
        beat = self.start_beat(
            logging_args=["-e", "-d", "*"],
            extra_args=["setup",
                        "--dashboards",
                        "-E", "setup.dashboards.file=" +
                        os.path.join(self.beat_path, "tests", "files", "testbeat-dashboards.zip"),
                        "-E", "setup.dashboards.beat=testbeat",
                        "-E", "setup.dashboards.only_index=true",
                        "-E", "setup.kibana.protocol=http",
                        "-E", "setup.kibana.host=" + self.get_kibana_host(),
                        "-E", "setup.kibana.port=" + self.get_kibana_port(),
                        "-E", "setup.kibana.username=beats",
                        "-E", "setup.kibana.password=testing",
                        "-E", "output.elasticsearch.hosts=['" + self.get_host() + "']",
                        "-E", "output.elasticsearch.username=admin",
                        "-E", "output.elasticsearch.password=testing",
                        "-E", "output.file.enabled=false"]
        )

        beat.check_wait(exit_code=0)

        assert self.log_contains("Kibana dashboards successfully loaded") is True

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_export_dashboard_cmd_export_dashboard_by_id(self):
        """
        Test testbeat export dashboard can export dashboards
        """
        self.render_config_template()
        self.test_load_dashboard()
        beat = self.start_beat(
            logging_args=["-e", "-d", "*"],
            extra_args=["export",
                        "dashboard",
                        "-E", "setup.kibana.protocol=http",
                        "-E", "setup.kibana.host=" + self.get_kibana_host(),
                        "-E", "setup.kibana.port=" + self.get_kibana_port(),
                        "-E", "setup.kibana.username=beats",
                        "-E", "setup.kibana.password=testing",
                        "-id", "Metricbeat-system-overview",
                        "-folder", "system-overview"]
        )

        beat.check_wait(exit_code=0)
        self._check_if_dashboard_exported("system-overview")

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_export_dashboard_cmd_export_dashboard_by_id_unknown_id(self):
        """
        Test testbeat export dashboard fails gracefully when dashboard with unknown ID is requested
        """
        self.render_config_template()
        beat = self.start_beat(
            logging_args=["-e", "-d", "*"],
            extra_args=["export",
                        "dashboard",
                        "-E", "setup.kibana.protocol=http",
                        "-E", "setup.kibana.host=" + self.get_kibana_host(),
                        "-E", "setup.kibana.port=" + self.get_kibana_port(),
                        "-E", "setup.kibana.username=beats",
                        "-E", "setup.kibana.password=testing",
                        "-id", "No-such-dashboard",
                        "-folder", "system-overview"]
        )

        beat.check_wait(exit_code=1)

        expected_error = re.compile("error exporting dashboard:.*not found", re.IGNORECASE)
        assert self.log_contains(expected_error)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_dev_tool_export_dashboard_by_id(self):
        """
        Test dev-tools/cmd/dashboards exports dashboard and removes unsupported characters
        """

        self.test_load_dashboard()

        folder_name = "system-overview"
        path = os.path.normpath(self.beat_path + "/../dev-tools/cmd/dashboards/export_dashboards.go")
        command = path + " -kibana http://" + self.get_kibana_host() + ":" + self.get_kibana_port()
        command = "go run " + command + " -dashboard Metricbeat-system-overview -folder " + folder_name

        p = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        content, err = p.communicate()
        assert p.returncode == 0

        self._check_if_dashboard_exported(folder_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_dev_tool_export_dashboard_by_id_unknown_id(self):
        """
        Test dev-tools/cmd/dashboards fails gracefully when dashboard with unknown ID is requested
        """

        path = os.path.normpath(self.beat_path + "/../dev-tools/cmd/dashboards/export_dashboards.go")
        command = path + " -kibana http://" + self.get_kibana_host() + ":" + self.get_kibana_port()
        command = "go run " + command + " -dashboard No-such-dashboard"

        p = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        content, err = p.communicate()

        assert p.returncode != 0

    def _check_if_dashboard_exported(self, folder_name):
        kibana_semver = semver.VersionInfo.parse(self.get_version())
        dashboard_folder = os.path.join(folder_name, "_meta", "kibana", str(kibana_semver.major), "dashboard")
        assert os.path.isdir(dashboard_folder)

        with open(os.path.join(dashboard_folder, "Metricbeat-system-overview.json")) as f:
            content = f.read()
            assert "Metricbeat-system-overview" in content

        shutil.rmtree(folder_name)

    def get_host(self):
        return os.getenv('ES_HOST', 'localhost') + ':' + os.getenv('ES_PORT', '9200')

    def get_kibana_host(self):
        return os.getenv('KIBANA_HOST', 'localhost')

    def get_kibana_port(self):
        return os.getenv('KIBANA_PORT', '5601')

    def get_version(self):
        url = "http://" + self.get_kibana_host() + ":" + self.get_kibana_port() + \
            "/api/status"

        r = requests.get(url, auth=("beats", "testing"))
        body = r.json()
        version = body["version"]["number"]

        return version
