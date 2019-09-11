# -*- coding: utf-8 -*-
from filebeat import BaseTest
import io
import os
import unittest

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

    def test_javascript_processor_add_locale(self):
        """
        Check JS processor with add_locale
        """

        self._test_javascript_processor_with_source("""\'var processor = require("processor");
var addLocale = new processor.AddLocale();
function process(evt) {
    addLocale.Run(evt);
}\'
""")

        output = self.read_output()
        for evt in output:
            assert "event.timezone" in evt

    def test_javascript_processor_add_process_metadata(self):
        """
        Check JS processor with add_process_metadata
        """

        self._test_javascript_processor_with_source("""\'var processor = require("processor");
var addProcessMetadata = new processor.AddProcessMetadata({
    match_pids: "process.pid",
    ignore_missing: false,
    overwrite_keys: true,
});
function process(evt) {
    addProcessMetadata.Run(evt);
}\'
""")

        output = self.read_output()
        for evt in output:
            assert "error.message" in evt
            assert evt["error.message"] == "GoError: none of the fields in match_pids found in the event"

    def test_javascript_processor_community_id(self):
        """
        Check JS processor with community_id
        """

        self._test_javascript_processor_with_source("""\'var processor = require("processor");
var communityID = new processor.CommunityID();
function process(evt) {
    communityID.Run(evt);
}\'
""", True)

        output = self.read_output()
        for evt in output:
            assert "network.community_id" in evt
            assert evt["network.community_id"] == "1:15+Ly6HsDg0sJdTmNktf6rko+os="

    def test_javascript_processor_copy_fields(self):
        """
        Check JS processor with copy_fields
        """

        self._test_javascript_processor_with_source("""\'var processor = require("processor");
var copy = new processor.CopyFields({
    fields: [
    {from: "message", to: "log.original"},
    ],
});
function process(evt) {
    copy.Run(evt);
}\'
""")

        output = self.read_output()
        for evt in output:
            assert "log.original" in evt
            assert evt["log.original"] == evt["message"]

    def test_javascript_processor_decode_json_fields(self):
        """
        Check JS processor with decode_json_fields
        """

        self._test_javascript_processor_with_source("""\'var processor = require("processor");
var decodeJSON = new processor.DecodeJSONFields({
    fields: ["json_message_to_decode"],
    target: "json_message",
});
function process(evt) {
    decodeJSON.Run(evt);
}\'
""", True)

        output = self.read_output()
        for evt in output:
            assert "json_message.key" in evt
            assert evt["json_message.key"] == "hello"

    def test_javascript_processor_dissect(self):
        """
        Check JS processor with dissect
        """

        self._test_javascript_processor_with_source("""\'var processor = require("processor");
var chopLog = new processor.Dissect({
    tokenizer: "key=%{key}",
    field: "dissect_message",
});
function process(evt) {
    chopLog.Run(evt);
}\'
""", True)

        output = self.read_output()
        for evt in output:
            assert "dissect.key" in evt
            assert evt["dissect.key"] == "hello"

    @unittest.skipIf(sys.platform == "nt", "Windows requires explicit DNS server configuration")
    def test_javascript_processor_dns(self):
        """
        Check JS processor with dns
        """

        self._test_javascript_processor_with_source("""\'var processor = require("processor");
var dns = new processor.DNS({
    type: "reverse",
    fields: {
        "source.ip": "source.domain",
        "destination.ip": "destination.domain"
    },
    tag_on_failure: ["_dns_reverse_lookup_failed"],
});
function process(evt) {
    dns.Run(evt);
    if (evt.Get().tags[0] !== "_dns_reverse_lookup_failed") {
        throw "missing tag";
    }
}\'
""", True)

    def test_javascript_processor_chain(self):
        """
        Check JS processor chain of processors
        """

        self._test_javascript_processor_with_source("""\'var processor = require("processor");
var localeProcessor = new processor.AddLocale();
var chain = new processor.Chain()
    .Add(localeProcessor)
    .Rename({
        fields: [
            {from: "event.timezone", to: "timezone"},
        ],
    })
    .Add(function(evt) {
        evt.Put("hello", "world");
    })
    .Build();

var chainOfChains = new processor.Chain()
    .Add(chain)
    .AddFields({fields: {foo: "bar"}})
    .Build();
function process(evt) {
    chainOfChains.Run(evt);
}\'
""")

        output = self.read_output()
        for evt in output:
            assert "timezone" in evt
            assert "hello" in evt
            assert evt["hello"] == "world"
            assert "fields.foo" in evt
            assert evt["fields.foo"] == "bar"

    def _test_javascript_processor_with_source(self, script_source, add_test_fields=False):
        js_proc = {
            "script": {
                "lang": "javascript",
                "source": script_source,
            },
        }

        # add dummy fields if configured
        processors = [js_proc]
        if add_test_fields:
            additional_fields = {
                "add_fields": {
                    "target": "''",
                    "fields": {
                        "source": {
                            "ip": "192.0.2.1",
                        },
                        "destination": {
                            "ip": "192.0.2.1",
                        },
                        "network": {
                            "transport": "igmp",
                        },
                        "dissect_message": "key=hello",
                        "json_message_to_decode": "{\"key\": \"hello\"}",
                    },
                }
            }
            processors = [additional_fields, js_proc]

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
            processors=processors,
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
