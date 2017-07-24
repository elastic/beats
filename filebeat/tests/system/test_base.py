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

    def test_invalid_config_with_removed_settings(self):
        """
        Checks if filebeat fails to load if removed settings have been used:
        """
        self.render_config_template(console={"pretty": "false"})

        exit_code = self.run_beat(extra_args=[
            "-E", "filebeat.spool_size=2048",
            "-E", "filebeat.publish_async=true",
            "-E", "filebeat.idle_timeout=1s",
        ])

        assert exit_code == 1
        assert self.log_contains("setting 'filebeat.spool_size'"
                                 " has been removed")
        assert self.log_contains("setting 'filebeat.publish_async'"
                                 " has been removed")
        assert self.log_contains("setting 'filebeat.idle_timeout'"
                                 " has been removed")
