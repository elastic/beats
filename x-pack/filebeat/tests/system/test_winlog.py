import jinja2
import requests
import platform
import sys
import hmac
import hashlib
import os
import json
import ast
from filebeat import BaseTest
from requests.auth import HTTPBasicAuth
import unittest


class Test(BaseTest):
    """
    Test filebeat with the winlog input
    """
    @classmethod
    def setUpClass(self):
        self.beat_name = "filebeat"
        self.beat_path = os.path.abspath(
            os.path.join(os.path.dirname(__file__), "../../"))

        super(BaseTest, self).setUpClass()

    def setUp(self):
        super(BaseTest, self).setUp()

        # Hack to make jinja2 have the right paths
        self.template_env = jinja2.Environment(
            loader=jinja2.FileSystemLoader([
                os.path.abspath(os.path.join(
                    self.beat_path, "../../filebeat")),
                os.path.abspath(os.path.join(self.beat_path, "../../libbeat"))
            ])
        )

    def get_config(self, options=None):
        """
        General function so that we do not have to define settings each time
        """
        evtx = os.path.join(os.path.dirname(__file__), "testdata", "1100.evtx")
        input_raw = """
- type: winlog
  enabled: true
  name: {} 
"""
        if options:
            input_raw = '\n'.join([input_raw, options])
        self.beat_name = "filebeat"
        self.beat_path = os.path.abspath(
            os.path.join(os.path.dirname(__file__), "../../"))

        input_raw = input_raw.format(evtx)
        self.render_config_template(
            input_raw=input_raw,
            inputs=False,
        )
        self.evtx = evtx

    def test_winlog_can_ingest(self):
        """
        Test winlog input with Windows event logs.
        """
        self.get_config()
        filebeat = self.start_beat()
        self.wait_until(lambda: self.log_contains("Input 'winlog' starting"))
        self.wait_until(lambda: self.output_count(lambda x: x >= 1))
        filebeat.check_kill_and_wait()

        output = self.read_output()

        assert output[0]["input.type"] == "winlog"
        assert output[0]["winlog.event_id"] == "1100"
        assert output[0]["message"] == "The event logging service has shut down."
