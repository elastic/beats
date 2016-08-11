from filebeat import BaseTest
import gzip
import os
import time
import unittest

"""
Tests that Filebeat is able to process gzipped log files.
"""

class Test(BaseTest):

    def test_gzipped_log_file(self):
        """
        Test expected event count for gzipped and regular log files
        """
        self.render_config_template(
                path=os.path.abspath(self.working_dir) + "/log/*",
                ignore_older="1h"
        )

        os.mkdir(self.working_dir + "/log/")
        expectedLineCount = 367
        self.copy_files(["logs/nasa-1.log.gz", "logs/nasa-360.log.gz", "logs/nasa-6.log"],
                        source_dir="../files",
                        target_dir="log")

        proc = self.start_beat()

        self.wait_until(
            lambda: self.output_has(lines=expectedLineCount),
            max_timeout=30)

        proc.check_kill_and_wait()

        objs = self.read_output()
        assert len(objs) == expectedLineCount
