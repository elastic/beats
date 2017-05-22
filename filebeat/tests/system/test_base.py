from filebeat import BaseTest
import os

"""
Test for basic object
"""


class Test(BaseTest):

    def test_base(self):
        """
        Test if the basic fields exist.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
        )

        with open(self.working_dir + "/test.log", "w") as f:
            f.write("test message\n")

        filebeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=1))
        filebeat.check_kill_and_wait()

        output = self.read_output()[0]
        assert "@timestamp" in output
        assert "prospector.type" in output
