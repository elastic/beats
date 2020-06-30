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
            "This is my super long line\n",
            "This is an even longer long line\n",
            "A végrehajtás során hiba történt\n",  # Error occured during execution (Hungarian)
            "This is OK\n",
        ])

        self._assert_expected_lines([
            "This is my",
            "This is an",
            "A végreha",
            "This is OK",
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
            "This is my super long line\n",
            "A végrehajtás során hiba történt\n",  # Error occured during execution (Hungarian)
            "This is OK\n",
        ])

        self._assert_expected_lines([
            "This is my",
            "A végrehaj",
            "This is OK",
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
            "42,\"string with \"\"quotes\"\"\"\n",
            ",\n"
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
            " 42\t hello world\t  \"string\twith tabs and \"broken\" quotes\"\n",
        ])

        self._assert_expected_lines([
            ["42", "hello world", "string\twith tabs and \"broken\" quotes"],
        ])

    def test_decode_csv_fields_header_in_string(self):
        """
        Check CSV decoding using header in string
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
            processors=[{
                "decode_csv_fields": {
                    "fields": {
                        "message": "csv"
                    },
                    "headers": {
                        "message": {
                            "string": "column1,column2,column3"
                        }
                    }
                },
            }]
        )

        self._init_and_read_test_input([
            "I,am,Mark\n"
        ])

        self._assert_expected_lines(["I"], field="csv.column1")

        self._assert_expected_lines(["am"], field="csv.column2")

        self._assert_expected_lines(["Mark"], field="csv.column3")

    def test_decode_csv_fields_header_in_file(self):
        """
        Check CSV decoding using header in conf file
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
            processors=[{
                "decode_csv_fields": {
                    "fields": {
                        "message": "csv"
                    },
                    "headers": {
                        "message": {
                            "file": {
                                "path": "tests/system/header_test"
                            }
                        }
                    }
                },
            }]
        )

        self._init_and_read_test_input([
            "I,am,Mark\n"
        ])

        self._assert_expected_lines(["I"], field="csv.column1_1")

        self._assert_expected_lines(["am"], field="csv.column1_2")

        self._assert_expected_lines(["Mark"], field="csv.column1_3")

    def test_decode_csv_fields_header_in_file_with_offset(self):
        """
        Check CSV decoding using header in conf file with offset
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
            processors=[{
                "decode_csv_fields": {
                    "fields": {
                        "message": "csv"
                    },
                    "headers": {
                        "message": {
                            "offset": 2,
                            "file": {
                                "path": "tests/system/header_test"
                            }
                        }
                    }
                },
            }]
        )

        self._init_and_read_test_input([
            "I,am,Mark\n"
        ])

        self._assert_expected_lines(["I"], field="csv.column2_1")

        self._assert_expected_lines(["am"], field="csv.column2_2")

        self._assert_expected_lines(["Mark"], field="csv.column2_3")

    def test_decode_csv_fields_header_in_file_offset_too_large(self):
        """
        Check CSV decoding using header in conf file with an offset too large
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
            processors=[{
                "decode_csv_fields": {
                    "fields": {
                        "message": "csv"
                    },
                    "headers": {
                        "message": {
                            "offset": 5,
                            "file": {
                                "path": "tests/system/header_test"
                            }
                        }
                    }
                },
            }]
        )
        with open(self.working_dir + "/test.log", "w") as f:
            f.write("column1,column2,column3\nI,am,Mark\n")

        filebeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=2))
        filebeat.check_kill_and_wait()

        output = self.read_output()[1]
        assert "csv.column1" not in output
        assert "csv.column2" not in output
        assert "csv.column3" not in output

    def test_decode_csv_fields_header_in_current_file(self):
        """
        Check CSV decoding using header in current file
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
            processors=[{
                "decode_csv_fields": {
                    "fields": {
                        "message": "csv"
                    },
                    "headers": {
                        "message": {
                            "in_file": True
                        }
                    }
                },
            }]
        )
        with open(self.working_dir + "/test.log", "w") as f:
            f.write("column1,column2,column3\nI,am,Mark\n")

        filebeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=2))
        filebeat.check_kill_and_wait()

        output = self.read_output()[1]
        assert output["csv.column1"] == "I"
        assert output["csv.column2"] == "am"
        assert output["csv.column3"] == "Mark"

    def test_urldecode_defaults(self):
        """
        Check URL-decoding using defaults
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
            processors=[{
                "urldecode": {
                    "fields": [{
                        "from": "message",
                        "to": "decoded"
                    }]
                },
            }]
        )

        self._init_and_read_test_input([
            "correct data\n",
            "correct%20data\n",
        ])

        self._assert_expected_lines([
            "correct data",
            "correct data",
        ], field="decoded")

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
            "test line 1\n",
            "test line 2\n",
            "test line 3\n",
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
