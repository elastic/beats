from filebeat import BaseTest
import gzip
import os
import time
import unittest

"""
Tests that Filebeat shuts down cleanly.
"""

class Test(BaseTest):

    def test_shutdown(self):
        """
        Test starting and stopping Filebeat under load.
        """

        # Uncompress the nasa log file.
        nasa_log = '../files/logs/nasa-50k.log'
        if not os.path.isfile(nasa_log):
            with gzip.open('../files/logs/nasa-50k.log.gz', 'rb') as infile:
                with open(nasa_log, 'w') as outfile:
                    for line in infile:
                        outfile.write(line)

        self.render_config_template(
                path=os.path.abspath(self.working_dir) + "/log/*",
                ignore_older="1h"
        )

        os.mkdir(self.working_dir + "/log/")
        self.copy_files(["logs/nasa-50k.log"],
                        source_dir="../files",
                        target_dir="log")

        for i in range(1,5):
            proc = self.start_beat(logging_args=["-e", "-v"])
            time.sleep(.5)
            proc.check_kill_and_wait()
