# coding=utf-8

from filebeat import BaseTest
import os

"""
Test Harvesters
"""


class Test(BaseTest):

    def test_input_type_deprecated(self):
        """
        Checks that harvesting works with deprecated input_type but message is outputted
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/test.log",
            input_type_deprecated="log",
            scan_frequency="0.1s"
        )
        os.mkdir(self.working_dir + "/log/")

        logfile = self.working_dir + "/log/test.log"

        with open(logfile, 'w') as f:
            f.write("Hello world\n")

        filebeat = self.start_beat()

        # Let it read the file
        self.wait_until(
            lambda: self.output_has(lines=1), max_timeout=10)

        filebeat.check_kill_and_wait()

        assert self.log_contains("DEPRECATED: input_type input config is deprecated")

    def test_prospectors_deprecated(self):
        """
        Checks that harvesting works with deprecated prospectors but a deprecation warning is printed.
        """

        self.render_config_template(
            input_config="prospectors",
            path=os.path.abspath(self.working_dir) + "/log/test.log",
            scan_frequency="0.1s"
        )
        os.mkdir(self.working_dir + "/log/")

        logfile = self.working_dir + "/log/test.log"

        with open(logfile, 'w') as f:
            f.write("Hello world\n")

        filebeat = self.start_beat()

        # Let it read the file
        self.wait_until(
            lambda: self.output_has(lines=1), max_timeout=10)

        filebeat.check_kill_and_wait()

        assert self.log_contains("DEPRECATED: prospectors are deprecated, Use `inputs` instead.")

    def test_reload_config_prospector_deprecated(self):
        """
        Checks that harvesting works with `config.prospectors`
        """

        inputConfigTemplate = """
        - type: log
          paths:
            - {}
          scan_frequency: 1s
        """

        self.render_config_template(
            reload_type="prospectors",
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

        assert self.log_contains("DEPRECATED: config.prospectors are deprecated, Use `config.inputs` instead.")
