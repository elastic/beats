# coding=utf-8

from filebeat import BaseTest
import os
import codecs
import time

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

        assert self.log_contains("DEPRECATED: input_type prospector config is deprecated")
