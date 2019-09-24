import os
import unittest
import platform
import socket
import sys
from heartbeat import BaseTest
from elasticsearch import Elasticsearch
from beat.beat import INTEGRATION_TESTS
import nose.tools
import logging
import subprocess
import time


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

        def has_started_message(): return self.log_contains("ICMP loop successfully initialized")

        def has_failed_message(): return self.log_contains("Failed to initialize ICMP loop")

        # We don't know if the system tests are running is configured to support or not support ping, but we can at least check that the ICMP loop
        # was initiated. In the future we should start up VMs with the correct perms configured and be more specific. In addition to that
        # we should run pings on those machines and make sure they work.
        self.wait_until(lambda: has_started_message() or has_failed_message(), 30)

        if has_failed_message():
            proc.check_kill_and_wait(1)
        else:
            # Check that documents are moving through
            self.wait_until(lambda: self.output_has(lines=1))
