from functionbeat import BaseTest

import os
import unittest


class Test(BaseTest):

    @unittest.skip("temporarily disabled")
    def test_base(self):
        """
        Basic test with exiting Functionbeat normally
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*"
        )

        functionbeat_proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("functionbeat is running"))
        exit_code = functionbeat_proc.kill_and_wait()
        assert exit_code == 0
