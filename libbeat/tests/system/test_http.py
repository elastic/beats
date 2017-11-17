from base import BaseTest

import requests
import json


class Test(BaseTest):

    def test_root(self):
        """
        Test / http endpoint
        """
        self.render_config_template(
        )

        proc = self.start_beat(extra_args=["-E", "http.enabled=true"])
        self.wait_until(lambda: self.log_contains("Starting stats endpoint"))

        r = requests.get("http://localhost:5066")
        assert r.status_code == 200

        data = json.loads(r.content)

        assert data["beat"] == "mockbeat"
        assert data["version"] == "9.9.9"

        proc.check_kill_and_wait()

    def test_stats(self):
        """
        Test /stats http endpoint
        """
        self.render_config_template(
        )

        proc = self.start_beat(extra_args=["-E", "http.enabled=true"])
        self.wait_until(lambda: self.log_contains("Starting stats endpoint"))

        r = requests.get("http://localhost:5066/stats")
        assert r.status_code == 200

        data = json.loads(r.content)

        # Test one data point
        assert data["libbeat"]["config"]["reloads"] == 0

        proc.check_kill_and_wait()

    def test_error(self):
        """
        Test not existing http endpoint
        """
        self.render_config_template(
        )

        proc = self.start_beat(extra_args=["-E", "http.enabled=true"])
        self.wait_until(lambda: self.log_contains("Starting stats endpoint"))

        r = requests.get("http://localhost:5066/not-exist")
        assert r.status_code == 404

        proc.check_kill_and_wait()
