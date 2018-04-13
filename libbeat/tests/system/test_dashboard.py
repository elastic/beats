from base import BaseTest
import os
import os.path
import subprocess
from nose.plugins.attrib import attr
import unittest


INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class Test(BaseTest):

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
    def test_export_dashboard(self):
        """
        Test export dashboards and remove unsupported characters
        """

        self.test_load_dashboard()

        command = self.beat_path + "/../dev-tools/cmd/dashboards/export_dashboards -kibana http://" + \
            self.get_kibana_host() + ":" + self.get_kibana_port()

        if os.name == "nt":
            command = self.beat_path + "\..\dev-tools\cmd\dashboards\export_dashboards -kibana http://" + \
                self.get_kibana_host() + ":" + self.get_kibana_port()

        command = command + " -dashboard Metricbeat-system-overview"

        print(command)

        p = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        content, err = p.communicate()

        assert p.returncode == 0

        assert os.path.isfile("output.json") is True

        with open('output.json') as f:
            content = f.read()
            assert "Metricbeat-system-overview" in content

        os.remove("output.json")

    def get_host(self):
        return os.getenv('ES_HOST', 'localhost') + ':' + os.getenv('ES_PORT', '9200')

    def get_kibana_host(self):
        return os.getenv('KIBANA_HOST', 'localhost')

    def get_kibana_port(self):
        return os.getenv('KIBANA_PORT', '5601')
