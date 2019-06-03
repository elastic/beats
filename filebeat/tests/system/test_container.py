from filebeat import BaseTest
import socket
import os


class Test(BaseTest):
    """
    Test filebeat with the container input
    """

    def test_container_input(self):
        """
        Test container input
        """
        input_raw = """
- type: container
  paths:
    - {}/logs/*.log
"""
        self.render_config_template(
            input_raw=input_raw.format(os.path.abspath(self.working_dir)),
            inputs=False,
        )

        os.mkdir(self.working_dir + "/logs/")
        self.copy_files(["logs/docker.log"],
                        target_dir="logs")

        filebeat = self.start_beat()

        self.wait_until(lambda:  self.output_has(lines=21))

        filebeat.check_kill_and_wait()

        output = self.read_output()
        assert len(output) == 21
        assert output[0]["message"] == "Fetching main repository github.com/elastic/beats..."
        for o in output:
            assert o["stream"] == "stdout"

    def test_container_input_cri(self):
        """
        Test container input with CRI format
        """
        input_raw = """
- type: container
  paths:
    - {}/logs/*.log
"""
        self.render_config_template(
            input_raw=input_raw.format(os.path.abspath(self.working_dir)),
            inputs=False,
        )

        os.mkdir(self.working_dir + "/logs/")
        self.copy_files(["logs/cri.log"],
                        target_dir="logs")

        filebeat = self.start_beat()

        self.wait_until(lambda: self.output_count(lambda x: x >= 1))
        self.wait_until(lambda: self.log_contains("End of file reached"))

        filebeat.check_kill_and_wait()

        output = self.read_output()
        assert len(output) == 1
        assert output[0]["stream"] == "stdout"
