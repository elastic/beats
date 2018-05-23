from filebeat import BaseTest
import os
import six

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
                        target_dir="log")

        proc = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)

        proc.check_kill_and_wait()

        output = self.read_output()[0]
        assert output["source"] == "hello"
        assert output["message"] == "test source"

    def test_json_add_tags(self):
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            json=dict(
                keys_under_root=True,
            ),
            agent_tags=["tag3", "tag4"]
        )
        os.mkdir(self.working_dir + "/log/")
        self.copy_files(["logs/json_tag.log"],
                        target_dir="log")

        proc = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)

        proc.check_kill_and_wait()

        output = self.read_output()[0]
        assert sorted(output["tags"]) == ["tag1", "tag2", "tag3", "tag4"]

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
            match="after",
            pattern="^\\["
        )

        proc = self.start_beat()
        status = proc.wait()
        assert status != 0
        assert self.log_contains("When using the JSON decoder and multiline" +
                                 " together, you need to specify a" +
                                 " message_key value")

    def test_timestamp_in_message(self):
        """
        Should be able to make use of a `@timestamp` field if it exists in the
        message.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            json=dict(
                message_key="msg",
                keys_under_root=True,
                overwrite_keys=True
            ),
        )
        os.mkdir(self.working_dir + "/log/")
        self.copy_files(["logs/json_timestamp.log"],
                        target_dir="log")

        proc = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=5),
            max_timeout=10)
        proc.check_kill_and_wait()

        output = self.read_output()
        assert len(output) == 5
        assert all(isinstance(o["@timestamp"], six.string_types) for o in output)
        assert output[0]["@timestamp"] == "2016-04-05T18:47:18.444Z"

        assert output[1]["@timestamp"] != "invalid"
        assert output[1]["error.message"] == \
            "@timestamp not overwritten (parse error on invalid)"

        assert output[2]["error.message"] == \
            "@timestamp not overwritten (not string)"

        assert "error" not in output[3]
        assert output[3]["@timestamp"] == "2016-04-05T18:47:18.444Z", output[3]["@timestamp"]

        assert "error" not in output[4]
        assert output[4]["@timestamp"] == "2016-04-05T18:47:18.000Z", output[4]["@timestamp"]

    def test_type_in_message(self):
        """
        If overwrite_keys is true and type is in the message, we have to
        be careful to keep it as a valid type name.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            json=dict(
                message_key="msg",
                keys_under_root=True,
                overwrite_keys=True
            ),
        )
        os.mkdir(self.working_dir + "/log/")
        self.copy_files(["logs/json_type.log"],
                        target_dir="log")

        proc = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=3),
            max_timeout=10)
        proc.check_kill_and_wait()

        output = self.read_output()
        assert len(output) == 3
        assert all(isinstance(o["@timestamp"], six.string_types) for o in output)
        assert output[0]["type"] == "test"

        assert "type" not in output[1]
        assert output[1]["error.message"] == \
            "type not overwritten (not string)"

        assert "type" not in output[2]
        assert output[2]["error.message"] == \
            "type not overwritten (not string)"

    def test_with_generic_filtering(self):
        """
        It should work fine to combine JSON decoding with
        removing fields via generic filtering. The test log file
        in here also contains a null value.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            json=dict(
                message_key="message",
                keys_under_root=True,
                overwrite_keys=True,
                add_error_key=True
            ),
            processors=[{
                "drop_fields": {
                    "fields": ["headers.request-id"],
                },
            }]
        )

        os.mkdir(self.working_dir + "/log/")
        self.copy_files(["logs/json_null.log"],
                        target_dir="log")

        proc = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)
        proc.check_kill_and_wait()

        output = self.read_output(
            required_fields=["@timestamp"],
        )
        assert len(output) == 1
        o = output[0]

        assert "headers.content-type" in o
        assert "headers.request-id" not in o

        # We drop null values during the generic event conversion.
        assert "res" not in o

    def test_json_decoding_error_true(self):
        """
        Test if json_decoding_error is set to true, that no errors are logged.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            json=dict(
                message_key="message",
                ignore_decoding_error=True
            ),
        )

        os.mkdir(self.working_dir + "/log/")

        testfile1 = self.working_dir + "/log/test.log"

        message = "invalidjson"
        with open(testfile1, 'a') as f:
            f.write(message + "\n")

        proc = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)
        proc.check_kill_and_wait()

        output = self.read_output(
            required_fields=["@timestamp"],
        )
        assert len(output) == 1
        assert output[0]["message"] == message
        assert False == self.log_contains_count("Error decoding JSON")

    def test_json_decoding_error_false(self):
        """
        Test if json_decoding_error is set to false, that an errors is logged.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            json=dict(
                message_key="message",
                ignore_decoding_error=False
            ),
        )

        os.mkdir(self.working_dir + "/log/")

        testfile1 = self.working_dir + "/log/test.log"

        message = "invalidjson"
        with open(testfile1, 'a') as f:
            f.write(message + "\n")

        proc = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)
        proc.check_kill_and_wait()

        output = self.read_output(
            required_fields=["@timestamp"],
        )
        assert len(output) == 1
        assert output[0]["message"] == message
        assert True == self.log_contains_count("Error decoding JSON")

    def test_with_generic_filtering_remove_headers(self):
        """
        It should work fine to combine JSON decoding with
        removing fields via generic filtering. The test log file
        in here also contains a null value.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            json=dict(
                message_key="message",
                keys_under_root=True,
                overwrite_keys=True,
                add_error_key=True
            ),
            processors=[{
                "drop_fields": {
                    "fields": ["headers", "res"],
                },
            }]
        )

        os.mkdir(self.working_dir + "/log/")
        self.copy_files(["logs/json_null.log"],
                        target_dir="log")

        proc = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)
        proc.check_kill_and_wait()

        output = self.read_output(
            required_fields=["@timestamp"],
        )
        assert len(output) == 1
        o = output[0]

        assert "headers.content-type" not in o
        assert "headers.request-id" not in o
        assert "res" not in o
        assert o["method"] == "GET"
        assert o["message"] == "Sent response: "

    def test_integer_condition(self):
        """
        It should work to drop JSON event based on an integer
        value by using a simple `equal` condition. See #2038.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            json=dict(
                keys_under_root=True,
            ),
            processors=[{
                "drop_event": {
                    "when": "equals.status: 200",
                },
            }]
        )
        os.mkdir(self.working_dir + "/log/")
        self.copy_files(["logs/json_int.log"],
                        target_dir="log")

        proc = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)
        proc.check_kill_and_wait()

        output = self.read_output()
        assert len(output) == 1
        assert output[0]["status"] == 404
