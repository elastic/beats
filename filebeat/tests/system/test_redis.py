from filebeat import BaseTest
from beat.beat import INTEGRATION_TESTS
import os
import unittest
import glob
import subprocess
import json
import logging
import redis


class Test(BaseTest):

    def init(self):
        r = redis.StrictRedis(host=self.get_host(), port=self.get_port())
        # Set to a very low value so every request falls under the slow log
        r.config_set('slowlog-log-slower-than', 1)
        r.config_set('slowlog-max-len', 200)
        return r

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    def test_input(self):
        r = self.init()
        r.set("hello", "world")

        input_raw = """
- type: redis
  hosts: ["{}:{}"]
  enabled: true
  scan_frequency: 1s
"""
        input_raw = input_raw.format(self.get_host(), self.get_port())

        self.render_config_template(
            input_raw=input_raw,
            inputs=False,
        )

        filebeat = self.start_beat()

        # wait for log to be read
        self.wait_until(lambda: self.output_count(lambda x: x >= 1))

        filebeat.check_kill_and_wait()

        output = self.read_output()[0]

        assert output["prospector.type"] == "redis"
        assert output["input.type"] == "redis"
        assert "redis.slowlog.cmd" in output

    def get_host(self):
        return os.getenv('REDIS_HOST', 'localhost')

    def get_port(self):
        return os.getenv('REDIS_PORT', '6379')
