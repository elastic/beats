from base import BaseTest

import requests
import json


class Test(BaseTest):
    def setUp(self):
        super(BaseTest, self).setUp()
        self.render_config_template()
        self.proc = self.start_beat(extra_args=["-E", "http.enabled=true", "-E", "http.pprof.enabled=true"])
        self.wait_until(lambda: self.log_contains("Starting stats endpoint"))

    def tearDown(self):
        super(BaseTest, self).tearDown()
        # Wait till the beat is completely started so it can handle SIGTERM
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.proc.check_kill_and_wait()

    def test_pprof(self):
        """
        Test /debug/pprof/ http endpoint
        """
        r = requests.get("http://localhost:5066/debug/pprof/")
        assert r.status_code == 200

    def test_pprof_cmdline(self):
        """
        Test /debug/pprof/cmdline http endpoint
        """
        r = requests.get("http://localhost:5066/debug/pprof/cmdline")
        assert r.status_code == 200

    def test_pprof_error(self):
        """
        Test not existing http endpoint
        """
        r = requests.get("http://localhost:5066/debug/pprof/not-exist")
        assert r.status_code == 404
