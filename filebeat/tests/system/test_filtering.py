from filebeat import BaseTest
import os

"""
Contains tests for filtering.
"""


class Test(BaseTest):
    def test_dropfields(self):
        """
        Check drop_fields filtering action
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
            filter_enabled=True,
            drop_fields=["beat"],
            include_fields=None,
        )
        with open(self.working_dir + "/test.log", "w") as f:
            f.write("test message\n")

        filebeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=1))
        filebeat.check_kill_and_wait()

        output = self.read_output(
            required_fields=["@timestamp", "type"],
        )[0]
        assert "beat.name" not in output
        assert "message" in output

    def test_include_fields(self):
        """
        Check drop_fields filtering action
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
            filter_enabled=True,
            include_fields=["source", "offset", "message"]
        )
        with open(self.working_dir + "/test.log", "w") as f:
            f.write("test message\n")

        filebeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=1))
        filebeat.check_kill_and_wait()

        output = self.read_output(
            required_fields=["@timestamp", "type"],
        )[0]
        assert "beat.name" not in output
        assert "message" in output
