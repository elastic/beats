import os
from filebeat import BaseTest, log_as_filestream


inputConfigTemplate = """
- type: log
  id: test-id
  allow_deprecated_use: true
  paths:
    - {}
  scan_frequency: 1s
"""


starting_msg = "Starting runner: input"
if log_as_filestream():
    starting_msg = "Starting runner: filestream"


class Test(BaseTest):

    def test_reload_same_input(self):
        """
        Test reloading same input
        """
        self.render_config_template(
            reload=True,
            reload_path=self.working_dir + "/configs/*.yml",
            inputs=False,
        )

        proc = self.start_beat()

        os.mkdir(self.working_dir + "/logs/")
        logfile = self.working_dir + "/logs/test.log"
        os.mkdir(self.working_dir + "/configs/")
        first_line = "First log file"
        second_line = "Second log file"

        config = inputConfigTemplate.format(self.working_dir + "/logs/test.log")
        config = config + """
  close_eof: true
"""
        with open(self.working_dir + "/configs/input.yml", 'w') as f:
            f.write(config)

        with open(logfile, 'w') as f:
            f.write(first_line + "\n")

        self.wait_until(lambda: self.output_lines() == 1)

        # Overwrite input with same path but new fields
        with open(self.working_dir + "/configs/input.yml", 'w') as f:
            config = config + """
  fields:
    hello: world
"""
            f.write(config)

        # Wait until input is stopped
        stopped_msg = "Runner: 'input [type=log]' has stopped"
        if log_as_filestream():
            stopped_msg = "Runner: 'filestream' has stopped"
        self.wait_until(
            lambda: self.log_contains(stopped_msg),
            max_timeout=15)

        # Update both log files, only 1 change should be picked up
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

    def test_load_configs(self):
        """
        Test loading separate inputs configs
        """
        self.render_config_template(
            reload_path=self.working_dir + "/configs/*.yml",
            inputs=False,
        )

        os.mkdir(self.working_dir + "/logs/")
        logfile = self.working_dir + "/logs/test.log"
        os.mkdir(self.working_dir + "/configs/")

        first_line = "First log file"
        second_line = "Second log file"

        config = inputConfigTemplate.format(self.working_dir + "/logs/test.log")
        config = config + """
  close_eof: true
"""
        with open(self.working_dir + "/configs/input.yml", 'w') as f:
            f.write(config)

        with open(logfile, 'w') as f:
            f.write(first_line + "\n")

        proc = self.start_beat()

        self.wait_until(lambda: self.output_lines() == 1)

        # Update both log files, only 1 change should be picked up
        with open(logfile, 'a') as f:
            f.write(second_line + "\n")

        self.wait_until(lambda: self.output_lines() == 2)

        proc.check_kill_and_wait()

        output = self.read_output()

        # Reloading stopped.
        self.wait_until(
            lambda: self.log_contains("Loading of config files completed."),
            max_timeout=15)

        # Make sure the correct lines were picked up
        assert self.output_lines() == 2
        assert output[0]["message"] == first_line
        assert output[1]["message"] == second_line
