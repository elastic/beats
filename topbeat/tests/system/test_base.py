from topbeat import BaseTest

import os
import shutil
import time

"""
Contains tests for base config
"""


class Test(BaseTest):
    def test_invalid_config(self):
        """
        Checks stop when input and topbeat defined
        """
        shutil.copy("./config/topbeat-input-invalid.yml",
                    os.path.join(self.working_dir, "invalid.yml"))

        exit_code = self.run_beat(config="invalid.yml", extra_args=["-N"])

        assert exit_code == 1
        assert self.log_contains("topbeat and input are set in config") is True

    def test_old_config(self):
        """
        Test that old config still works with deprecation warning
        """
        shutil.copy("./config/topbeat-old.yml",
                    os.path.join(self.working_dir, "topbeat-old.yml"))

        topbeat = self.start_beat(config="topbeat-old.yml", extra_args=["-N"])
        time.sleep(1)
        topbeat.check_kill_and_wait()

        assert self.log_contains("The 'input' configuration section is deprecated. Please use 'topbeat' instead") is True

