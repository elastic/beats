from filebeat import BaseTest
import os

"""
Tests for the JSON decoding functionality.
"""


class Test(BaseTest):
    def test_docker_logs(self):
        """
        Should be able to interpret docker logs.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            json=dict(message_key="log")
        )

        os.mkdir(self.working_dir + "/log/")
        self.copy_files(["logs/docker.log"],
                        source_dir="../files",
                        target_dir="log")

        proc = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=21),
            max_timeout=10)

        proc.check_kill_and_wait()

        output = self.read_output()
        assert len(output) == 21
        assert all("json.log" in o for o in output)
        assert all("json.time" in o for o in output)
        assert all(o["json.stream"] == "stdout" for o in output)

    def test_docker_logs_filtering(self):
        """
        Should be able to do line filtering on docker logs.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            json=dict(message_key="log", keys_under_root=True),
            exclude_lines=["windows"]
        )

        os.mkdir(self.working_dir + "/log/")
        self.copy_files(["logs/docker.log"],
                        source_dir="../files",
                        target_dir="log")

        proc = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=19),
            max_timeout=10)

        proc.check_kill_and_wait()

        output = self.read_output()
        assert len(output) == 19

        assert all("log" in o for o in output)
        assert all("time" in o for o in output)
        assert all(o["stream"] == "stdout" for o in output)
        assert all("windows" not in o["log"] for o in output)

    def test_docker_logs_multiline(self):
        """
        Should be able to do multiline on docker logs.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            json=dict(message_key="log", keys_under_root=True),
            multiline=True,
            pattern="^\[log\]",
            match="after",
            negate="true"
        )

        os.mkdir(self.working_dir + "/log/")
        self.copy_files(["logs/docker_multiline.log"],
                        source_dir="../files",
                        target_dir="log")

        proc = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=3),
            max_timeout=10)

        proc.check_kill_and_wait()

        output = self.read_output()
        assert len(output) == 3

        assert all("time" in o for o in output)
        assert all("log" in o for o in output)
        assert all("message" not in o for o in output)
        assert all(o["stream"] == "stdout" for o in output)
        assert output[1]["log"] == \
            "[log] This one is\n on multiple\n lines"

    def test_simple_json_overwrite(self):
        """
        Should be able to overwrite keys when requested.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            json=dict(
                message_key="message",
                keys_under_root=True,
                overwrite_keys=True),
            exclude_lines=["windows"]
        )

        os.mkdir(self.working_dir + "/log/")
        self.copy_files(["logs/json_override.log"],
                        source_dir="../files",
                        target_dir="log")

        proc = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)

        proc.check_kill_and_wait()

        output = self.read_output()[0]
        assert output["source"] == "hello"
        assert output["message"] == "test source"

    def test_config_no_msg_key_filtering(self):
        """
        Should raise an error if line filtering and JSON are defined,
        but the message key is not defined.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            json=dict(
                keys_under_root=True),
            exclude_lines=["windows"]
        )

        proc = self.start_beat()
        status = proc.wait()
        assert status != 0
        assert self.log_contains("When using the JSON decoder and line" +
                                 " filtering together, you need to specify" +
                                 " a message_key value")

    def test_config_no_msg_key_multiline(self):
        """
        Should raise an error if line filtering and JSON are defined,
        but the message key is not defined.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            json=dict(
                keys_under_root=True),
            multiline=True,
            pattern="^["
        )

        proc = self.start_beat()
        status = proc.wait()
        assert status != 0
        assert self.log_contains("When using the JSON decoder and multiline" +
                                 " together, you need to specify a" +
                                 " message_key value")
