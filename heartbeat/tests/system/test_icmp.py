import logging
import os
import platform
import socket
import subprocess
import sys
import time
import unittest
import re
from beat.beat import INTEGRATION_TESTS
from elasticsearch import Elasticsearch
from heartbeat import BaseTest


class Test(BaseTest):
    def test_base(self):
        """
        Basic test with icmp root non privilege ICMP test.

        """

        config = {
            "monitors": [
                {
                    "type": "icmp",
                    "schedule": "*/5 * * * * * *",
                    "hosts": ["127.0.0.1"],
                }
            ]
        }

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            **config
        )

        proc = self.start_beat()

        # because we have no way of knowing if the current environment has the ability to do ICMP pings
        # we are instead asserting the monitor's status via the output and checking for errors where appropriate
        self.wait_until(lambda: self.output_has(lines=1))
        output = self.read_output()
        monitor_status = output[0]["monitor.status"]
        assert monitor_status == "up" or monitor_status == "down"
        assert output[0]["monitor.type"] == "icmp"
