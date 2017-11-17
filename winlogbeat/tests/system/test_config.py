import sys
import unittest
from winlogbeat import BaseTest

"""
Contains tests for config parsing.
"""


@unittest.skipUnless(sys.platform.startswith("win"), "requires Windows")
class Test(BaseTest):

    def test_valid_config(self):
        """
        configtest - valid config
        """
        self.render_config_template(
            ignore_older="1h",
            event_logs=[
                {"name": "Application", "ignore_older": "48h"}
            ]
        )
        self.start_beat(extra_args=["-configtest"]).check_wait()

    def test_invalid_ignore_older(self):
        """
        configtest - invalid ignore_older units (1 hour)
        """
        self.render_config_template(
            event_logs=[
                {"name": "Application", "ignore_older": "1 hour"}
            ]
        )
        self.start_beat(extra_args=["-configtest"]).check_wait(exit_code=1)
        assert self.log_contains(
            "unknown unit  hour in duration 1 hour "
            "accessing 'winlogbeat.event_logs.0.ignore_older'")

    def test_invalid_level(self):
        """
        configtest - invalid level (errors)
        """
        self.render_config_template(
            event_logs=[
                {"name": "Application", "level": "errors"}
            ]
        )
        self.start_beat(extra_args=["-configtest"]).check_wait(exit_code=1)
        assert self.log_contains(
            "invalid level ('errors') for query")

    def test_invalid_api(self):
        """
        configtest - invalid api (file)
        """
        self.render_config_template(
            event_logs=[
                {"name": "Application", "api": "file"}
            ]
        )
        self.start_beat(extra_args=["-configtest"]).check_wait(exit_code=1)
        assert self.log_contains("Failed to create new event log. "
                                 "file API is not available")
