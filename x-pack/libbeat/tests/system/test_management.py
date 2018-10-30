import sys
import os

from base import BaseTest


INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class TestManagement(BaseTest):

    def test_enroll(self):
        """
        Enroll the beat in Kibana Central Management
        """

        config_path = os.path.join(self.working_dir, "mockbeat.yml")
        self.render_config_template("mockbeat", config_path)

        config_content = open(config_path, 'r').read()

        exit_code = self.run_beat(
            extra_args=["enroll", self.get_kibana_url(),
                        "--password", "env:PASS", "--force"],
            logging_args=["-v", "-d", "*"],
            env={
                'PASS': 'changeme',
            })

        assert exit_code == 0
        assert self.log_contains("Enrolled and ready to retrieve settings")

        # Enroll creates a keystore (to store access token)
        assert os.path.isfile(os.path.join(
            self.working_dir, "mockbeat.keystore"))

        # Settings backup has been created
        assert os.path.isfile(os.path.join(
            self.working_dir, "mockbeat.yml.bak"))
        backup_content = open(config_path + ".bak", 'r').read()
        assert config_content == backup_content

    def get_kibana_url(self):
        return 'http://' + os.getenv('KIBANA_HOST', 'kibana') + ':' + os.getenv('KIBANA_PORT', '5601')
