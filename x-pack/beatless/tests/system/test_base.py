from beatless import BaseTest

import os
import unittest


class Test(BaseTest):

    @unittest.skip("temporarily disabled")
    def test_base(self):
        """
        Basic test with exiting Beatless normally
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*"
        )

        beatless_proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("beatless is running"))
        exit_code = beatless_proc.kill_and_wait()
        assert exit_code == 0
