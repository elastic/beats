from base import BaseTest

import requests
import json


class Test(BaseTest):
    def setUp(self):
        super(BaseTest, self).setUp()
        self.render_config_template()
        self.proc = self.start_beat(extra_args=["-E", "http.enabled=true"])
        self.wait_until(lambda: self.log_contains("Starting stats endpoint"))

    def tearDown(self):
        super(BaseTest, self).tearDown()
        # Wait till the beat is completely started so it can handle SIGTERM
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.proc.check_kill_and_wait()

    def test_root(self):
        """
        Test / http endpoint
        """
        r = requests.get("http://localhost:5066")
        assert r.status_code == 200

        data = json.loads(r.content.decode('utf_8'))

        assert data["beat"] == "mockbeat"
        assert data["version"] == "9.9.9"

    def test_stats(self):
        """
        Test /stats http endpoint
        """
        r = requests.get("http://localhost:5066/stats")
        assert r.status_code == 200

        data = json.loads(r.content.decode('utf_8'))

        # Test one data point
        assert data["libbeat"]["config"]["scans"] == 0

    def test_error(self):
        """
        Test not existing http endpoint
        """
        r = requests.get("http://localhost:5066/not-exist")
        assert r.status_code == 404

    def test_pprof_disabled(self):
        """
        Test /debug/pprof/ http endpoint
        """
        r = requests.get("http://localhost:5066/debug/pprof/")
        assert r.status_code == 404
