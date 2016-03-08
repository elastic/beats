from filebeat import BaseTest
import os

"""
Tests for generic filtering
"""


class Test(BaseTest):

    def test_drop_fields(self):

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
            filter_enabled=True,
            drop_fields=["source"],
        )

        with open(self.working_dir + "/test.log", "w") as f:
            f.write("test message\n")

        filebeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=1))
        filebeat.kill_and_wait()

        output = self.read_output()

        print output

        assert len(output) > 0
        assert "source" not in output[0]
