import re
import sys
import unittest
import os
import time
from filebeat import BaseTest


prospectorConfigTemplate = """
- input_type: log
  paths:
    - {}
  scan_frequency: 1s
"""


class Test(BaseTest):

    def test_reload(self):
        """
        Test basic reload
        """
        self.render_config_template(
            reload=True,
            reload_path=self.working_dir + "/configs/*.yml",
            prospectors=False,
        )

        proc = self.start_beat()

        os.mkdir(self.working_dir + "/logs/")
        logfile = self.working_dir + "/logs/test.log"
        os.mkdir(self.working_dir + "/configs/")

        with open(self.working_dir + "/configs/prospector.yml", 'w') as f:
            f.write(prospectorConfigTemplate.format(self.working_dir + "/logs/*"))

        with open(logfile, 'w') as f:
            f.write("Hello world\n")

        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()

    def test_start_stop(self):
        """
        Test basic start and stop
        """
        self.render_config_template(
            reload=True,
            reload_path=self.working_dir + "/configs/*.yml",
            prospectors=False,
        )

        proc = self.start_beat()

        os.mkdir(self.working_dir + "/logs/")
        logfile = self.working_dir + "/logs/test.log"
        os.mkdir(self.working_dir + "/configs/")

        with open(self.working_dir + "/configs/prospector.yml", 'w') as f:
            f.write(prospectorConfigTemplate.format(self.working_dir + "/logs/*"))

        with open(logfile, 'w') as f:
            f.write("Hello world\n")

        self.wait_until(lambda: self.output_lines() == 1)

        # Remove prospector
        with open(self.working_dir + "/configs/prospector.yml", 'w') as f:
            f.write("")

        # Wait until prospector is stopped
        self.wait_until(
            lambda: self.log_contains("Runner stopped:"),
            max_timeout=15)

        with open(logfile, 'a') as f:
            f.write("Hello world\n")

        # Wait to give a change to pick up the new line (it shouldn't)
        time.sleep(1)

        proc.check_kill_and_wait()

        assert self.output_lines() == 1

    def test_start_stop_replace(self):
        """
        Test basic start and replace with an other prospecto
        """
        self.render_config_template(
            reload=True,
            reload_path=self.working_dir + "/configs/*.yml",
            prospectors=False,
        )

        proc = self.start_beat()

        os.mkdir(self.working_dir + "/logs/")
        logfile1 = self.working_dir + "/logs/test1.log"
        logfile2 = self.working_dir + "/logs/test2.log"
        os.mkdir(self.working_dir + "/configs/")
        first_line = "First log file"
        second_line = "Second log file"

        with open(self.working_dir + "/configs/prospector.yml", 'w') as f:
            f.write(prospectorConfigTemplate.format(self.working_dir + "/logs/test1.log"))

        with open(logfile1, 'w') as f:
            f.write(first_line + "\n")

        self.wait_until(lambda: self.output_lines() == 1)

        # Remove prospector
        with open(self.working_dir + "/configs/prospector.yml", 'w') as f:
            f.write("")

        # Wait until prospector is stopped
        self.wait_until(
            lambda: self.log_contains("Runner stopped:"),
            max_timeout=15)

        with open(self.working_dir + "/configs/prospector.yml", 'w') as f:
            f.write(prospectorConfigTemplate.format(self.working_dir + "/logs/test2.log"))

        # Update both log files, only 1 change should be picke dup
        with open(logfile1, 'a') as f:
            f.write("First log file 1\n")
        with open(logfile2, 'a') as f:
            f.write(second_line + "\n")

        self.wait_until(lambda: self.output_lines() == 2)

        proc.check_kill_and_wait()

        output = self.read_output()

        # Make sure the correct lines were picked up
        assert output[0]["message"] == first_line
        assert output[1]["message"] == second_line
        assert self.output_lines() == 2

    def test_reload_same_prospector(self):
        """
        Test reloading same prospector
        """
        self.render_config_template(
            reload=True,
            reload_path=self.working_dir + "/configs/*.yml",
            prospectors=False,
        )

        proc = self.start_beat()

        os.mkdir(self.working_dir + "/logs/")
        logfile = self.working_dir + "/logs/test.log"
        os.mkdir(self.working_dir + "/configs/")
        first_line = "First log file"
        second_line = "Second log file"

        config = prospectorConfigTemplate.format(self.working_dir + "/logs/test.log")
        config = config + """
  close_eof: true
"""
        with open(self.working_dir + "/configs/prospector.yml", 'w') as f:
            f.write(config)

        with open(logfile, 'w') as f:
            f.write(first_line + "\n")

        self.wait_until(lambda: self.output_lines() == 1)

        # Overwrite prospector with same path but new fields
        with open(self.working_dir + "/configs/prospector.yml", 'w') as f:
            config = config + """
  fields:
    hello: world
"""
            f.write(config)

        # Wait until prospector is stopped
        self.wait_until(
            lambda: self.log_contains("Runner stopped:"),
            max_timeout=15)

        # Update both log files, only 1 change should be picke dup
        with open(logfile, 'a') as f:
            f.write(second_line + "\n")

        self.wait_until(lambda: self.output_lines() == 2)

        proc.check_kill_and_wait()

        output = self.read_output()

        # Make sure the correct lines were picked up
        assert self.output_lines() == 2
        assert output[0]["message"] == first_line
        # Check no fields exist
        assert ("fields" in output[0]) == False
        assert output[1]["message"] == second_line
        # assert that fields are added
        assert output[1]["fields.hello"] == "world"
