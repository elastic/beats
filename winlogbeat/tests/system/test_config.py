import os
import subprocess
import sys
import unittest
from winlogbeat import BaseTest
from beat import common_tests

"""
Contains tests for config parsing.
"""


@unittest.skipUnless(sys.platform.startswith("win"), "requires Windows")
class Test(BaseTest, common_tests.TestExportsMixin):

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
        self.run_config_tst(exit_code=0)

    def test_invalid_ignore_older(self):
        """
        configtest - invalid ignore_older units (1 hour)
        """
        self.render_config_template(
            event_logs=[
                {"name": "Application", "ignore_older": "1 hour"}
            ]
        )
        self.run_config_tst(exit_code=1)
        assert self.log_contains(
            "unknown unit \" hour\" in duration \"1 hour\" "
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
        self.run_config_tst(exit_code=1)
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
        self.run_config_tst(exit_code=1)
        assert self.log_contains("failed to create new event log: "
                                 "file API is not available")

    def run_config_tst(self, pcap=None, exit_code=0):
        config = "winlogbeat.yml"

        cmd = os.path.join(self.beat_path, "winlogbeat.test")
        args = [
            cmd, "-systemTest",
            "-c", os.path.join(self.working_dir, config),
        ]

        if os.getenv("TEST_COVERAGE") == "true":
            args += [
                "-test.coverprofile",
                os.path.join(self.working_dir, "coverage.cov"),
            ]

        args.extend(["test", "config"])

        output = "winlogbeat-" + self.today + ".ndjson"

        with open(os.path.join(self.working_dir, output), "wb") as outfile:
            proc = subprocess.Popen(args,
                                    stdout=outfile,
                                    stderr=subprocess.STDOUT
                                    )
            actual_exit_code = proc.wait()

        if actual_exit_code != exit_code:
            print("============ Log Output =====================")
            with open(os.path.join(self.working_dir, output)) as f:
                print(f.read())
            print("============ Log End Output =====================")
        assert actual_exit_code == exit_code, "Expected exit code to be %d, but it was %d" % (
            exit_code, actual_exit_code)
        return actual_exit_code
