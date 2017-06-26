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

    def test_monitor_config(self):
        """
        Basic test with fields and tags in monitor
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/*"
        )

        heartbeat_proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        heartbeat_proc.check_kill_and_wait()

        doc = self.read_output()[0]
        assert doc["hello"] == "world"
        assert doc["tags"] == ["http_monitor_tags"]
        assert "fields" not in doc
