from filebeat import BaseTest
import os
import socket

"""
Tests for the custom fields functionality.
"""


class Test(BaseTest):

    def test_custom_fields(self):
        """
        Tests that custom fields show up in the output dict.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
            fields={"hello": "world", "number": 2}
        )

        with open(self.working_dir + "/test.log", "w") as f:
            f.write("test message\n")

        filebeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=1))
        filebeat.check_kill_and_wait()

        output = self.read_output()
        doc = output[0]
        assert doc["fields.hello"] == "world"
        assert doc["fields.number"] == 2

    def test_custom_fields_under_root(self):
        """
        Tests that custom fields show up in the output dict under
        root when fields_under_root option is used.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
            fields={
                "hello": "world",
                "type": "log2",
                "timestamp": "2"
            },
            fieldsUnderRoot=True
        )

        with open(self.working_dir + "/test.log", "w") as f:
            f.write("test message\n")

        filebeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=1))
        filebeat.check_kill_and_wait()

        output = self.read_output()
        doc = output[0]
        print(doc)
        assert doc["hello"] == "world"
        assert doc["type"] == "log2"
        assert doc["timestamp"] == 2
        assert "fields" not in doc

    def test_beat_fields(self):
        """
        Checks that it's possible to set a custom shipper name. Also
        tests that beat.hostname  has values.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
            shipper_name="testShipperName"
        )

        with open(self.working_dir + "/test.log", "w") as f:
            f.write("test message\n")

        filebeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=1))
        filebeat.check_kill_and_wait()

        output = self.read_output()
        doc = output[0]
        assert doc["host.name"] == "testShipperName"
        assert doc["agent.hostname"] == socket.gethostname()
        assert "fields" not in doc
