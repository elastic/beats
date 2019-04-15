#!/usr/bin/env python

from filebeat import BaseTest
import os

from beat.beat import Proc

"""
Tests for the stdin functionality.
"""


class Test(BaseTest):

    def test_stdin(self):
        """
        Test stdin input. Checks if reading is continued after the first read.
        """
        self.render_config_template(
            type="stdin"
        )

        proc = self.start_beat()

        self.wait_until(
            lambda: self.log_contains(
                "Harvester started for file: -"),
            max_timeout=10)

        iterations1 = 5
        for n in range(0, iterations1):
            os.write(proc.stdin_write, "Hello World\n")

        self.wait_until(
            lambda: self.output_has(lines=iterations1),
            max_timeout=15)

        iterations2 = 10
        for n in range(0, iterations2):
            os.write(proc.stdin_write, "Hello World\n")

        self.wait_until(
            lambda: self.output_has(lines=iterations1 + iterations2),
            max_timeout=15)

        proc.check_kill_and_wait()

        objs = self.read_output()
        assert len(objs) == iterations1 + iterations2

    def test_stdin_eof(self):
        """
        Test that Filebeat works when stdin is closed.
        """
        self.render_config_template(
            type="stdin",
            close_eof="true",
        )

        args = [self.test_binary, "-systemTest"]
        if os.getenv("TEST_COVERAGE") == "true":
            args += ["-test.coverprofile",
                     os.path.join(self.working_dir, "coverage.cov")]
        args += ["-c", os.path.join(self.working_dir, "filebeat.yml"), "-e",
                 "-v", "-d", "*"]
        proc = Proc(args, os.path.join(self.working_dir, "filebeat.log"))
        os.write(proc.stdin_write, "Hello World\n")

        proc.start()
        self.wait_until(lambda: self.output_has(lines=1))

        # Continue writing after end was reached
        os.write(proc.stdin_write, "Hello World2\n")
        os.close(proc.stdin_write)

        self.wait_until(lambda: self.output_has(lines=2))

        proc.proc.terminate()
        proc.proc.wait()

        objs = self.read_output()
        assert objs[0]["message"] == "Hello World"
        assert objs[1]["message"] == "Hello World2"

    def test_stdin_is_exclusive(self):
        """
        Test that Filebeat run Stdin in exclusive mode.
        """

        input_raw = """
- type: stdin
  enabled: true
- type: udp
  host: 127.0.0.0:10000
  enabled: true
"""

        self.render_config_template(
            input_raw=input_raw,
            inputs=False,
        )

        filebeat = self.start_beat()
        filebeat.check_wait(exit_code=1)
        assert self.log_contains("Exiting: stdin requires to be run in exclusive mode, configured inputs: stdin, udp")
