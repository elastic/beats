import os
import time
from filebeat import BaseTest


inputConfigTemplate = """
- type: log
  paths:
    - {}
  scan_frequency: 1s
"""


class Test(BaseTest):

    def test_reload(self):
        """
        Test basic input reload
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

        with open(self.working_dir + "/configs/input.yml", 'w') as f:
            f.write(inputConfigTemplate.format(self.working_dir + "/logs/*"))

        with open(logfile, 'w') as f:
            f.write("Hello world\n")

        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()

    def test_start_stop(self):
        """
        Test basic input start and stop
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

        with open(self.working_dir + "/configs/input.yml", 'w') as f:
            f.write(inputConfigTemplate.format(self.working_dir + "/logs/*"))

        with open(logfile, 'w') as f:
            f.write("Hello world\n")

        self.wait_until(lambda: self.output_lines() == 1)

        # Remove input
        with open(self.working_dir + "/configs/input.yml", 'w') as f:
            f.write("")

        # Wait until input is stopped
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
        Test basic start and replace with another input
        """
        self.render_config_template(
            reload=True,
            reload_path=self.working_dir + "/configs/*.yml",
            inputs=False,
        )

        proc = self.start_beat()

        os.mkdir(self.working_dir + "/logs/")
        logfile1 = self.working_dir + "/logs/test1.log"
        logfile2 = self.working_dir + "/logs/test2.log"
        os.mkdir(self.working_dir + "/configs/")
        first_line = "First log file"
        second_line = "Second log file"

        with open(self.working_dir + "/configs/input.yml", 'w') as f:
            f.write(inputConfigTemplate.format(self.working_dir + "/logs/test1.log"))

        with open(logfile1, 'w') as f:
            f.write(first_line + "\n")

        self.wait_until(lambda: self.output_lines() == 1)

        # Remove input
        with open(self.working_dir + "/configs/input.yml", 'w') as f:
            f.write("")

        # Wait until input is stopped
        self.wait_until(
            lambda: self.log_contains("Runner stopped:"),
            max_timeout=15)

        with open(self.working_dir + "/configs/input.yml", 'w') as f:
            f.write(inputConfigTemplate.format(self.working_dir + "/logs/test2.log"))

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

        # Update both log files, only 1 change should be picke dup
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

    def test_reload_same_config(self):
        """
        Test reload same config with same file but different config. Makes sure reloading also works on conflicts.
        """
        self.render_config_template(
            reload=True,
            reload_path=self.working_dir + "/configs/*.yml",
            inputs=False,
        )

        os.mkdir(self.working_dir + "/logs/")
        logfile = self.working_dir + "/logs/test.log"
        os.mkdir(self.working_dir + "/configs/")

        with open(self.working_dir + "/configs/input.yml", 'w') as f:
            f.write(inputConfigTemplate.format(self.working_dir + "/logs/*"))

        proc = self.start_beat()

        with open(logfile, 'w') as f:
            f.write("Hello world1\n")

        self.wait_until(lambda: self.output_lines() > 0)

        # New config with same config file but a bit different to make it reload
        # Add it intentionally when other input is still running to cause an error
        with open(self.working_dir + "/configs/input.yml", 'w') as f:
            f.write(inputConfigTemplate.format(self.working_dir + "/logs/test.log"))

        # Make sure error shows up in log file
        self.wait_until(
            lambda: self.log_contains("Can only start an input when all related states are finished"),
            max_timeout=15)

        # Wait until old runner is stopped
        self.wait_until(
            lambda: self.log_contains("Runner stopped:"),
            max_timeout=15)

        # Add new log line and see if it is picked up = new input is running
        with open(logfile, 'a') as f:
            f.write("Hello world2\n")

        self.wait_until(lambda: self.output_lines() > 1)

        proc.check_kill_and_wait()

    def test_reload_add(self):
        """
        Test adding a input and makes sure both are still running
        """
        self.render_config_template(
            reload=True,
            reload_path=self.working_dir + "/configs/*.yml",
            inputs=False,
        )

        os.mkdir(self.working_dir + "/logs/")
        logfile1 = self.working_dir + "/logs/test1.log"
        logfile2 = self.working_dir + "/logs/test2.log"
        os.mkdir(self.working_dir + "/configs/")

        with open(self.working_dir + "/configs/input.yml", 'w') as f:
            f.write(inputConfigTemplate.format(self.working_dir + "/logs/test1.log"))

        proc = self.start_beat()

        with open(logfile1, 'w') as f:
            f.write("Hello world1\n")

        self.wait_until(lambda: self.output_lines() > 0)

        with open(self.working_dir + "/configs/input2.yml", 'w') as f:
            f.write(inputConfigTemplate.format(self.working_dir + "/logs/test2.log"))

        self.wait_until(
            lambda: self.log_contains_count("New runner started") == 2,
            max_timeout=15)

        # Add new log line and see if it is picked up = new input is running
        with open(logfile1, 'a') as f:
            f.write("Hello world2\n")

        # Add new log line and see if it is picked up = new input is running
        with open(logfile2, 'a') as f:
            f.write("Hello world3\n")

        self.wait_until(lambda: self.output_lines() == 3)

        proc.check_kill_and_wait()
