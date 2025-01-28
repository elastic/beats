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
  allow_deprecated_use: true
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

        self.wait_until(lambda: self.output_has(lines=21))

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
  allow_deprecated_use: true
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

    def test_container_input_registry_for_unparsable_lines(self):
        """
        Test container input properly updates registry offset in case
        of unparsable lines
        """
        input_raw = """
- type: container
  allow_deprecated_use: true
  paths:
    - {}/logs/*.log
"""
        self.render_config_template(
            input_raw=input_raw.format(os.path.abspath(self.working_dir)),
            inputs=False,
        )

        os.mkdir(self.working_dir + "/logs/")
        self.copy_files(["logs/docker_corrupted.log"],
                        target_dir="logs")

        filebeat = self.start_beat()

        self.wait_until(lambda: self.output_has(lines=20))

        filebeat.check_kill_and_wait()

        output = self.read_output()
        assert len(output) == 20
        assert output[19]["message"] == "Moving binaries to host..."
        for o in output:
            assert o["stream"] == "stdout"

        # Check that file exist
        data = self.get_registry()
        logs = self.log_access()
        assert logs.contains("Parse line error") == True
        # bytes of healthy file are 2244 so for the corrupted one should
        # be 2244-1=2243 since we removed one character
        assert data[0]["offset"] == 2243
