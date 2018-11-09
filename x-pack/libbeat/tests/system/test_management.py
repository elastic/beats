import sys
import os
import json
import requests

from base import BaseTest


INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)
KIBANA_PASSWORD = 'changeme'


class TestManagement(BaseTest):

    def test_enroll(self):
        """
        Enroll the beat in Kibana Central Management
        """
        # We don't care about this as it will be replaced by enrollment
        # process:
        config_path = os.path.join(self.working_dir, "mockbeat.yml")
        self.render_config_template("mockbeat", config_path)

        config_content = open(config_path, 'r').read()

        exit_code = self.enroll(KIBANA_PASSWORD)

        assert exit_code == 0
        assert self.log_contains("Enrolled and ready to retrieve settings")

        # Enroll creates a keystore (to store access token)
        assert os.path.isfile(os.path.join(
            self.working_dir, "mockbeat.keystore"))

        # New settings file is in place now
        new_content = open(config_path, 'r').read()
        assert config_content != new_content

        # Settings backup has been created
        assert os.path.isfile(os.path.join(
            self.working_dir, "mockbeat.yml.bak"))
        backup_content = open(config_path + ".bak", 'r').read()
        assert config_content == backup_content

    def test_enroll_bad_pw(self):
        """
        Try to enroll the beat in Kibana Central Management with a bad password
        """
        # We don't care about this as it will be replaced by enrollment
        # process:
        config_path = os.path.join(self.working_dir, "mockbeat.yml")
        self.render_config_template("mockbeat", config_path)

        config_content = open(config_path, 'r').read()

        exit_code = self.enroll('wrong password')

        assert exit_code == 1

        # Keystore wasn't created
        assert not os.path.isfile(os.path.join(
            self.working_dir, "mockbeat.keystore"))

        # Settings hasn't changed
        new_content = open(config_path, 'r').read()
        assert config_content == new_content

    def test_fetch_configs(self):
        """
        Config is retrieved from Central Management and updates are applied
        """
        # Enroll the beat
        config_path = os.path.join(self.working_dir, "mockbeat.yml")
        self.render_config_template("mockbeat", config_path)
        exit_code = self.enroll(KIBANA_PASSWORD)
        assert exit_code == 0

        # Configure an output
        self.create_and_assing_tag([
            {
                "type": "output",
                "configs": [
                    {
                        "output": "elasticsearch",
                        "elasticsearch": {
                            "hosts": ["localhost:9200"],
                            "username":"elastic",
                            "password": KIBANA_PASSWORD,
                        }
                    }
                ]
            }
        ])

        # Start beat
        proc = self.start_beat(extra_args=["-E", "management.period=1s"])

        # Wait for beat to apply new conf
        self.wait_log_contains("Applying settings for output")

        # Update output configuration
        self.create_and_assing_tag([
            {
                "type": "output",
                "configs": [
                    {
                        "output": "file",
                        "file": {
                            "path": os.path.join(self.working_dir, "output"),
                            "filename": "mockbeat",
                        }
                    }
                ]
            }
        ])

        # Wait for beat to apply new conf, now it logs to console
        self.wait_until(
            cond=lambda: self.log_contains_count("Applying settings for output") == 2)

        self.wait_until(cond=lambda: self.output_has(1))

        proc.check_kill_and_wait()

    def test_configs_cache(self):
        """
        Config cache is used if Kibana is not available
        """
        # Enroll the beat
        config_path = os.path.join(self.working_dir, "mockbeat.yml")
        self.render_config_template("mockbeat", config_path)
        exit_code = self.enroll(KIBANA_PASSWORD)
        assert exit_code == 0

        # Update output configuration
        self.create_and_assing_tag([
            {
                "type": "output",
                "configs": [
                    {
                        "output": "file",
                        "file": {
                            "path": os.path.join(self.working_dir, "output"),
                            "filename": "mockbeat_managed",
                        }
                    }
                ]
            }
        ])

        output_file = os.path.join("output", "mockbeat_managed")

        # Start beat
        proc = self.start_beat()
        self.wait_until(cond=lambda: self.output_has(
            1, output_file=output_file))
        proc.check_kill_and_wait()

        # Remove output file
        os.remove(os.path.join(self.working_dir, output_file))

        # Cache should exists already, start with wrong kibana settings:
        proc = self.start_beat(extra_args=[
            "-E", "management.kibana.host=wronghost",
            "-E", "management.kibana.timeout=0.5s",
        ])
        self.wait_until(cond=lambda: self.output_has(
            1, output_file=output_file))
        proc.check_kill_and_wait()

    def enroll(self, password):
        return self.run_beat(
            extra_args=["enroll", self.get_kibana_url(),
                        "--password", "env:PASS", "--force"],
            logging_args=["-v", "-d", "*"],
            env={
                'PASS': password,
            })

    def create_and_assing_tag(self, blocks):
        tag_name = "test"
        headers = {
            "kbn-xsrf": "1"
        }

        # Create tag
        url = self.get_kibana_url() + "/api/beats/tag/" + tag_name
        data = {
            "color": "#DD0A73",
            "configuration_blocks": blocks,
        }

        r = requests.put(url, json=data, headers=headers,
                         auth=('elastic', KIBANA_PASSWORD))
        assert r.status_code in (200, 201)

        # Retrieve beat UUID
        meta = json.loads(
            open(os.path.join(self.working_dir, 'data', 'meta.json'), 'r').read())

        # Assign tag to beat
        data = {"assignments": [{"beatId": meta["uuid"], "tag": tag_name}]}
        url = self.get_kibana_url() + "/api/beats/agents_tags/assignments"
        r = requests.post(url, json=data, headers=headers,
                          auth=('elastic', KIBANA_PASSWORD))
        assert r.status_code == 200

    def get_kibana_url(self):
        return 'http://' + os.getenv('KIBANA_HOST', 'kibana') + ':' + os.getenv('KIBANA_PORT', '5601')
