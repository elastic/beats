import os

from heartbeat import BaseTest


class Test(BaseTest):

    def test_base(self):
        """
        Basic test with exiting Heartbeat normally
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*"
        )

        heartbeat_proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("heartbeat is running"))
        heartbeat_proc.check_kill_and_wait()
