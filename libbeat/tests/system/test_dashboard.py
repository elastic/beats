from base import BaseTest
import os
import os.path
import subprocess
from nose.plugins.attrib import attr
import unittest
from unittest import SkipTest
import requests
import semver

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class Test(BaseTest):

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
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
                        "-E", "output.elasticsearch.hosts=['" + self.get_host() + "']",
                        "-E", "output.file.enabled=false"]
        )

        beat.check_wait(exit_code=0)

        assert self.log_contains("Skipping loading dashboards")

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
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
                        "-E", "output.elasticsearch.hosts=['" + self.get_host() + "']",
                        "-E", "output.file.enabled=false"]
        )

        beat.check_wait(exit_code=0)

        assert self.log_contains("Kibana dashboards successfully loaded") is True

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_load_dashboard_into_space(self, create_space=True):
        """
        Test loading dashboards into Kibana space
        """
        version = self.get_version()
        if semver.compare(version, "6.5.0") == -1:
            # Skip for Kibana versions < 6.5.0 as Kibana Spaces not available
            raise SkipTest

        self.render_config_template()
        if create_space:
            self.create_kibana_space()

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
                        "-E", "setup.kibana.space.id=foo-bar",
                        "-E", "output.elasticsearch.hosts=['" + self.get_host() + "']",
                        "-E", "output.file.enabled=false"]
        )

        beat.check_wait(exit_code=0)

        assert self.log_contains("Kibana dashboards successfully loaded") is True

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
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
                        "-E", "output.elasticsearch.hosts=['" + self.get_host() + "']",
                        "-E", "output.file.enabled=false"]
        )

        beat.check_wait(exit_code=0)

        assert self.log_contains("Kibana dashboards successfully loaded") is True

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_export_dashboard_cmd_export_dashboard_by_id_and_decoding(self):
        """
        Test testbeat export dashboard can export dashboards
        and removes unsupported characters
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
                        "-decode",
                        "-id", "Metricbeat-system-overview"]
        )

        beat.check_wait(exit_code=0)

        assert self.log_contains("\"id\": \"Metricbeat-system-overview\",") is True

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
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
                        "-id", "Metricbeat-system-overview"]
        )

        beat.check_wait(exit_code=0)

        assert self.log_contains("\"id\": \"Metricbeat-system-overview\",") is True

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
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
                        "-id", "No-such-dashboard"]
        )

        beat.check_wait(exit_code=1)

        assert self.log_contains("error exporting dashboard: Not found") is True

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_export_dashboard_cmd_export_dashboard_from_yml(self):
        """
        Test testbeat export dashboard can export dashboards from dashboards YAML file
        and removes unsupported characters
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
                        "-yml", os.path.join(self.beat_path, "tests", "files", "dashboards.yml")]
        )

        beat.check_wait(exit_code=0)

        version = self.get_version()
        kibana_semver = semver.VersionInfo.parse(version)
        exported_dashboard_path = os.path.join(self.beat_path, "tests", "files", "_meta",
                                               "kibana", str(kibana_semver.major), "dashboard", "Metricbeat-system-test-overview.json")

        with open(exported_dashboard_path) as f:
            content = f.read()
            assert "Metricbeat-system-overview" in content

        os.remove(exported_dashboard_path)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_export_dashboard_cmd_export_dashboard_from_not_existent_yml(self):
        """
        Test testbeat export dashboard fails gracefully when cannot find YAML file
        """

        self.render_config_template()
        beat = self.start_beat(
            logging_args=["-e", "-d", "*"],
            extra_args=["export",
                        "dashboard",
                        "-E", "setup.kibana.protocol=http",
                        "-E", "setup.kibana.host=" + self.get_kibana_host(),
                        "-E", "setup.kibana.port=" + self.get_kibana_port(),
                        "-yml", os.path.join(self.beat_path, "tests", "files", "no-such-file.yml")]
        )

        beat.check_wait(exit_code=1)
        assert self.log_contains("Error exporting dashboards from yml")
        assert self.log_contains("error opening the list of dashboards")

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_dev_tool_export_dashboard_by_id(self):
        """
        Test dev-tools/cmd/dashboards exports dashboard and removes unsupported characters
        """

        self.test_load_dashboard()

        path = os.path.normpath(self.beat_path + "/../dev-tools/cmd/dashboards/export_dashboards.go")
        command = path + " -kibana http://" + self.get_kibana_host() + ":" + self.get_kibana_port()
        command = "go run " + command + " -dashboard Metricbeat-system-overview"

        p = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        content, err = p.communicate()

        assert p.returncode == 0

        assert os.path.isfile("output.json") is True

        with open('output.json') as f:
            content = f.read()
            assert "Metricbeat-system-overview" in content

        os.remove("output.json")

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
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

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_dev_tool_export_dashboard_by_id_from_space(self):
        """
        Test dev-tools/cmd/dashboards exports dashboard from Kibana space
        and removes unsupported characters
        """
        version = self.get_version()
        if semver.compare(version, "6.5.0") == -1:
            # Skip for Kibana versions < 6.5.0 as Kibana Spaces not available
            raise SkipTest

        self.test_load_dashboard_into_space(False)

        path = os.path.normpath(self.beat_path + "/../dev-tools/cmd/dashboards/export_dashboards.go")
        command = path + " -kibana http://" + self.get_kibana_host() + ":" + self.get_kibana_port()
        command = "go run " + command + " -dashboard Metricbeat-system-overview -space-id foo-bar"

        p = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        content, err = p.communicate()

        assert p.returncode == 0

        assert os.path.isfile("output.json") is True

        with open('output.json') as f:
            content = f.read()
            assert "Metricbeat-system-overview" in content

        os.remove("output.json")

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_dev_tool_export_dashboard_from_yml(self):
        """
        Test dev-tools/cmd/dashboards exports dashboard from dashboards YAML file
        and removes unsupported characters
        """

        self.test_load_dashboard()

        path = os.path.normpath(self.beat_path + "/../dev-tools/cmd/dashboards/export_dashboards.go")
        command = path + " -kibana http://" + self.get_kibana_host() + ":" + self.get_kibana_port()
        command = "go run " + command + " -yml " + os.path.join(self.beat_path, "tests", "files", "dashboards.yml")

        p = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        content, err = p.communicate()

        assert p.returncode == 0

        version = self.get_version()
        kibana_semver = semver.VersionInfo.parse(version)
        exported_dashboard_path = os.path.join(self.beat_path, "tests", "files", "_meta",
                                               "kibana", str(kibana_semver.major), "dashboard", "Metricbeat-system-test-overview.json")

        with open(exported_dashboard_path) as f:
            content = f.read()
            assert "Metricbeat-system-overview" in content

        os.remove(exported_dashboard_path)

    def get_host(self):
        return os.getenv('ES_HOST', 'localhost') + ':' + os.getenv('ES_PORT', '9200')

    def get_kibana_host(self):
        return os.getenv('KIBANA_HOST', 'localhost')

    def get_kibana_port(self):
        return os.getenv('KIBANA_PORT', '5601')

    def create_kibana_space(self):
        url = "http://" + self.get_kibana_host() + ":" + self.get_kibana_port() + \
            "/api/spaces/space"
        data = {
            "id": "foo-bar",
            "name": "Foo bar space"
        }

        headers = {
            "kbn-xsrf": "1"
        }

        r = requests.post(url, json=data, headers=headers)
        assert r.status_code == 200

    def get_version(self):
        url = "http://" + self.get_kibana_host() + ":" + self.get_kibana_port() + \
            "/api/status"

        r = requests.get(url)
        body = r.json()
        version = body["version"]["number"]

        return version
