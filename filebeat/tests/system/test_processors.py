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
            processors=[{
                "drop_fields": {
                    "fields": ["beat"],
                },
            }]
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
            processors=[{
                "include_fields": {
                    "fields": ["source", "offset", "message"],
                },
            }]
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

    def test_drop_event(self):
        """
        Check drop_event filtering action
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test*.log",
            processors=[{
                "drop_event": {
                    "when": "contains.source: test1",
                },
            }]
        )
        with open(self.working_dir + "/test1.log", "w") as f:
            f.write("test1 message\n")

        with open(self.working_dir + "/test2.log", "w") as f:
            f.write("test2 message\n")

        filebeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=1))
        filebeat.check_kill_and_wait()

        output = self.read_output(
            required_fields=["@timestamp", "type"],
        )[0]
        assert "beat.name" in output
        assert "message" in output
        assert "test" in output["message"]

    def test_condition(self):
        """
        Check condition in processors
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test*.log",
            processors=[{
                "drop_event": {
                    "when": "not.contains.message: test",
                },
            }]
        )
        with open(self.working_dir + "/test1.log", "w") as f:
            f.write("test1 message\n")

        with open(self.working_dir + "/test2.log", "w") as f:
            f.write("test2 message\n")

        filebeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=2))
        filebeat.check_kill_and_wait()

        output = self.read_output(
            required_fields=["@timestamp", "type"],
        )[0]
        assert "beat.name" in output
        assert "message" in output
        assert "test" in output["message"]
