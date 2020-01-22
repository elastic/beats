from filebeat import BaseTest

import os
import platform
import time
import shutil
import json
from nose.plugins.skip import Skip, SkipTest


# Additional tests: to be implemented
# * Check if registrar file can be configured, set config param
# * Check "updating" of registrar file
# * Check what happens when registrar file is deleted


class Test(BaseTest):

    def test_registrar_file_content(self):
        """
        Check if registrar file is created correctly and content is as expected
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
        )
        os.mkdir(self.working_dir + "/log/")

        filebeat = self.start_beat()

        testfile = self.working_dir + "/log/test.log"
        file = open(testfile, 'w')

        iterations = 5
        for n in range(0, iterations):
            file.write("line " + str(n + 1))
            file.write("\n")

        file.close()

        # Let it read the file
        self.wait_until(
            lambda: self.output_has(lines=iterations), max_timeout=10)

        self.wait_until(
            lambda: self.output_has(lines=iterations), max_timeout=10)

        # Wait until registry file is written
        self.wait_until(
            lambda: self.log_contains_count(
                "Registry file updated.") > 1,
            max_timeout=15)

        filebeat.check_kill_and_wait()

        data = self.get_registry()
        assert len(data) == 1
        assert self.output_has(lines=iterations)
