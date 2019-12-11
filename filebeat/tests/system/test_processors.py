# -*- coding: utf-8 -*-
from filebeat import BaseTest
import io
import os
import unittest
import sys

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
                    "fields": ["agent"],
                },
            }]
        )
        with open(self.working_dir + "/test.log", "w") as f:
            f.write("test message\n")

        filebeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=1))
        filebeat.check_kill_and_wait()

        output = self.read_output(
            required_fields=["@timestamp"],
        )[0]
        assert "agent.type" not in output
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
            required_fields=["@timestamp"],
        )[0]
        assert "agent.type" not in output
        assert "message" in output

    def test_drop_event(self):
        """
        Check drop_event filtering action
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test*.log",
            processors=[{
                "drop_event": {
                    "when": "contains.log.file.path: test1",
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
            required_fields=["@timestamp"],
        )[0]
        assert "agent.type" in output
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
            required_fields=["@timestamp"],
        )[0]
        assert "agent.type" in output
        assert "message" in output
        assert "test" in output["message"]

    def test_dissect_good_tokenizer(self):
        """
        Check dissect with a good tokenizer
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
            processors=[{
                "dissect": {
                    "tokenizer": "\"%{key} world\"",
                    "field": "message",
                    "target_prefix": "extracted"
                },
            }]
        )
        with open(self.working_dir + "/test.log", "w") as f:
            f.write("Hello world\n")

        filebeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=1))
        filebeat.check_kill_and_wait()

        output = self.read_output(
            required_fields=["@timestamp"],
        )[0]
        assert output["extracted.key"] == "Hello"

    def test_dissect_defaults(self):
        """
        Check dissect defaults
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
            processors=[{
                "dissect": {
                    "tokenizer": "\"%{key} world\"",
                },
            }]
        )
        with open(self.working_dir + "/test.log", "w") as f:
            f.write("Hello world\n")

        filebeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=1))
        filebeat.check_kill_and_wait()

        output = self.read_output(
            required_fields=["@timestamp"],
        )[0]
        assert output["dissect.key"] == "Hello"

    def test_dissect_bad_tokenizer(self):
        """
        Check dissect with a bad tokenizer
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
            processors=[{
                "dissect": {
                    "tokenizer": "\"not %{key} world\"",
                    "field": "message",
                    "target_prefix": "extracted"
                },
            }]
        )
        with open(self.working_dir + "/test.log", "w") as f:
            f.write("Hello world\n")

        filebeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=1))
        filebeat.check_kill_and_wait()

        output = self.read_output(
            required_fields=["@timestamp"],
        )[0]
        assert "extracted.key" not in output
        assert output["message"] == "Hello world"

    def test_truncate_bytes(self):
        """
        Check if truncate_fields with max_bytes can truncate long lines and leave short lines as is
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
            processors=[{
                "truncate_fields": {
                    "max_bytes": 10,
                    "fields": ["message"],
                },
            }]
        )

        self._init_and_read_test_input([
            u"This is my super long line\n",
            u"This is an even longer long line\n",
            u"A végrehajtás során hiba történt\n",  # Error occured during execution (Hungarian)
            u"This is OK\n",
        ])

        self._assert_expected_lines([
            u"This is my",
            u"This is an",
            u"A végreha",
            u"This is OK",
        ])

    def test_truncate_characters(self):
        """
        Check if truncate_fields with max_charaters can truncate long lines and leave short lines as is
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
            processors=[{
                "truncate_fields": {
                    "max_characters": 10,
                    "fields": ["message"],
                },
            }]
        )

        self._init_and_read_test_input([
            u"This is my super long line\n",
            u"A végrehajtás során hiba történt\n",  # Error occured during execution (Hungarian)
            u"This is OK\n",
        ])

        self._assert_expected_lines([
            u"This is my",
            u"A végrehaj",
            u"This is OK",
        ])

    def test_decode_csv_fields_defaults(self):
        """
        Check CSV decoding using defaults
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
            processors=[{
                "decode_csv_fields": {
                    "fields": {
                        "message": "csv"
                    }
                },
            }]
        )

        self._init_and_read_test_input([
            u"42,\"string with \"\"quotes\"\"\"\n",
            u",\n"
        ])

        self._assert_expected_lines([
            ["42", "string with \"quotes\""],
            ["", ""]
        ], field="csv")

    def test_decode_csv_fields_all_options(self):
        """
        Check CSV decoding with options
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
            processors=[{
                "decode_csv_fields": {
                    "fields": {
                        "message": "message"
                    },
                    "overwrite_keys": True,
                    "separator": "\"\t\"",
                    "trim_leading_space": True,
                },
            }]
        )

        self._init_and_read_test_input([
            u" 42\t hello world\t  \"string\twith tabs and \"broken\" quotes\"\n",
        ])

        self._assert_expected_lines([
            ["42", "hello world", "string\twith tabs and \"broken\" quotes"],
        ])

    def test_javascript_processor_add_host_metadata(self):
        """
        Check JS processor with add_host_metadata
        """

        self._test_javascript_processor_with_source("""\'var processor = require("processor");
var addHostMetadata = new processor.AddHostMetadata({"netinfo.enabled": true});

function process(evt) {
    addHostMetadata.Run(evt);
}\'
""")

        output = self.read_output()
        for evt in output:
            assert "host.hostname" in evt

    def _test_javascript_processor_with_source(self, script_source):
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
            processors=[
                {
                    "script": {
                        "lang": "javascript",
                        "source": script_source,
                    },
                },
            ]
        )

        self._init_and_read_test_input([
            u"test line 1\n",
            u"test line 2\n",
            u"test line 3\n",
        ])

    def _init_and_read_test_input(self, input_lines):
        with io.open(self.working_dir + "/test.log", "w", encoding="utf-8") as f:
            for line in input_lines:
                f.write((line))

        filebeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=len(input_lines)))
        filebeat.check_kill_and_wait()

    def _assert_expected_lines(self, expected_lines, field="message"):
        output = self.read_output()

        assert len(output) == len(expected_lines)

        for i in range(len(expected_lines)):
            assert output[i][field] == expected_lines[i]
