import jinja2
import requests
import platform
import pytest
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
    Test filebeat with the etw input
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
        input_raw = """
- type: etw
  enabled: true
  provider.name: "Microsoft-Windows-Kernel-Process"
  session_name: TestSession
  match_any_keyword: 0xffffffffffffffff
  trace_level: verbose
"""
        if options:
            input_raw = '\n'.join([input_raw, options])
        self.beat_name = "filebeat"
        self.beat_path = os.path.abspath(
            os.path.join(os.path.dirname(__file__), "../../"))

        self.render_config_template(
            input_raw=input_raw,
            inputs=False,
        )

    @pytest.mark.skipif(sys.platform != "win32", reason="This test is specific to Windows")
    def test_etw_can_ingest(self):
        """
        Test ETW input with Windows trace logs
        """
        self.get_config()
        filebeat = self.start_beat()
        self.wait_until(lambda: self.log_contains("Input 'etw' starting"))
        self.wait_until(lambda: self.output_count(lambda x: x >= 1))
        filebeat.check_kill_and_wait()

        output = self.read_output()

        assert output[0]["input.type"] == "etw"
        assert output[0]["event.kind"] == "event"
        assert output[0]["winlog.provider_message"] == "Microsoft-Windows-Kernel-Process"
