import re
import sys
import metricbeat
import unittest


@unittest.skipUnless(re.match("(?i)win|linux|darwin|freebsd", sys.platform), "os")
class Test(metricbeat.BaseTest):
    """
    Test is the implementation of unit tests for processessors
    """

    def test_drop_fields(self):

        self.render_config_template(
            modules=[{
                "name": "system",
                "metricsets": ["network"],
                "period": "1s"
            }],
            processors=[{
                "drop_fields": {
                    "when": "range.system.cpu.system.pct.lt: 0.1",
                    "fields": ["system.network.in"],
                },
            }]
        )
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()

        output = self.read_output_json()
        self.assertGreater(len(output), 1)
        evt = output[0]
        self.assert_fields_are_documented(evt)

        print(evt)
        print(list(evt.keys()))
        self.assertCountEqual(self.de_dot([
            'agent', '@timestamp', 'system', 'metricset.module',
            'metricset.rtt', 'metricset.name', 'host', 'service', 'ecs', 'event'
        ]), evt.keys())
        network = evt["system"]["network"]
        print(list(network.keys()))
        self.assertCountEqual(self.de_dot(
            ["name", "out", "in"]), network.keys())

    def test_dropfields_with_condition(self):
        """
        Check drop_fields action works when a condition is associated.
        """
        self.render_config_template(
            modules=[{
                "name": "system",
                "metricsets": ["process"],
                "period": "1s",
                "processes": ["(?i)metricbeat.test"]
            }],
            processors=[{
                "drop_fields": {
                    "fields": ["system.process.memory"],
                },
            }]
        )
        metricbeat = self.start_beat()
        self.wait_until(
            lambda: self.output_count(lambda x: x >= 4),
            max_timeout=15)

        metricbeat.kill_and_wait()

        output = self.read_output(
            required_fields=["@timestamp"],
        )
        good_event = {}
        for evt in output:
            if "system.process.cpu.total.pct" in evt:
                good_event = evt

        if float(good_event["system.process.cpu.total.pct"]) < 0.5:
            assert "system.process.memory.size" not in good_event
        else:
            assert "system.process.memory.size" in good_event

    def test_dropevent_with_condition(self):
        """
        Check drop_event action works when a condition is associated.
        """
        self.render_config_template(
            modules=[{
                "name": "system",
                "metricsets": ["process"],
                "period": "1s",
                "processes": ["(?i)metricbeat.test"],
                "process.include_top_n": {"enabled": True, "by_cpu": 1}
            }],
            processors=[{
                "drop_event": {
                    "when": "range.system.process.cpu.total.pct.lt: 0.001",
                },
            }]
        )
        metricbeat = self.start_beat()
        self.wait_until(
            lambda: self.output_count(lambda x: x >= 4),
            max_timeout=20)

        metricbeat.kill_and_wait()

        output = self.read_output(
            required_fields=["@timestamp"],
        )
        good_event = {}
        for evt in output:
            if "system.process.cpu.total.pct" in evt:
                good_event = evt

        print(output)
        assert float(good_event["system.process.cpu.total.pct"]) >= 0.001

    def test_dropevent_with_complex_condition(self):
        """
        Check drop_event action works when a complex condition is associated.
        """
        self.render_config_template(
            modules=[{
                "name": "system",
                "metricsets": ["process"],
                "period": "1s"
            }],
            processors=[{
                "drop_event": {
                    "when.not": "contains.process.command_line: metricbeat.test",
                },
            }]
        )
        metricbeat = self.start_beat()
        self.wait_until(
            lambda: self.output_count(lambda x: x >= 1),
            max_timeout=15)

        metricbeat.kill_and_wait()

        output = self.read_output(
            required_fields=["@timestamp"],
        )
        assert len(output) >= 1

    def test_include_fields(self):
        """
        Check include_fields filtering action
        """
        self.render_config_template(
            modules=[{
                "name": "system",
                "metricsets": ["process"],
                "period": "1s",
                # filter down to one process, since this test suite doesn't have a way to specify X iterations of the metricbeat fetcher
                "processes": ["(?i)metricbeat.test"]
            }],
            processors=[{
                "include_fields": {"fields": ["system.process.cpu", "system.process.memory"]},
            }]
        )
        metricbeat = self.start_beat()
        self.wait_until(
            lambda: self.output_count(lambda x: x >= 4),
            max_timeout=15)

        metricbeat.kill_and_wait()

        output = self.read_output(
            required_fields=["@timestamp"],
        )
        good_event = {}
        # depending on threading, or whatever is happening, try to search for a fully formatted event
        for evt in output:
            if "system.process.cpu.total.pct" in evt:
                good_event = evt
        print(output)

        for key in [
            "system.process.cpu.start_time",
            "system.process.cpu.total.pct",
            "system.process.memory.size",
            "system.process.memory.rss.bytes",
            "system.process.memory.rss.pct"
        ]:
            assert key in good_event

        for key in [
            "system.process.name",
            "system.process.pid",
        ]:
            assert key not in good_event

    def test_multiple_actions(self):
        """
        Check the result when configuring two actions: include_fields
        and drop_fields.
        """
        self.render_config_template(
            modules=[{
                "name": "system",
                "metricsets": ["process"],
                "period": "1s",
                "processes": ["(?i)metricbeat.test"],
            }],
            processors=[{
                "include_fields": {"fields": ["system.process", "process"]},
            }, {
                "drop_fields": {"fields": ["system.process.memory"]},
            }]
        )
        metricbeat = self.start_beat()
        self.wait_until(
            lambda: self.output_count(lambda x: x >= 4),
            max_timeout=15)

        metricbeat.kill_and_wait()

        output = self.read_output(
            required_fields=["@timestamp"],
        )
        good_event = {}

        for evt in output:
            if "system.process.cpu.total.pct" in evt:
                good_event = evt
        print(output)

        for key in [
            "system.process.cpu.start_time",
            "system.process.cpu.total.pct",
            "process.name",
            "process.pid",
        ]:
            assert key in good_event, "'%s' not found" % key

        for key in [
            "system.process.memory.size",
            "system.process.memory.rss.bytes",
            "system.process.memory.rss.pct"
        ]:
            assert key not in good_event, "'%s' not expected but found" % key

    def test_contradictory_multiple_actions(self):
        """
        Check the behaviour of a contradictory multiple actions
        """
        self.render_config_template(
            modules=[{
                "name": "system",
                "metricsets": ["process"],
                "period": "1s"
            }],
            processors=[{
                "include_fields": {
                    "fields": ["system.process.memory.size", "proc.memory.rss.pct"],
                },
            }, {
                "drop_fields": {
                    "fields": ["system.process.memory.size", "proc.memory.rss.pct"],
                },
            }]
        )
        metricbeat = self.start_beat()
        self.wait_until(
            lambda: self.output_count(lambda x: x >= 1),
            max_timeout=15)
        metricbeat.kill_and_wait()

        output = self.read_output(
            required_fields=["@timestamp"],
        )[0]

        for key in [
            "system.process.memory.size",
            "system.process.memory.rss",
            "system.process.cpu.start_time",
            "system.process.cpu.total.pct",
            "system.process.name",
            "system.process.pid",
            "system.process.memory.rss.pct"
        ]:
            assert key not in output

    def test_rename_field(self):

        self.render_config_template(
            modules=[{
                "name": "system",
                "metricsets": ["cpu"],
                "period": "1s"
            }],
            processors=[{
                "rename": {
                    "fields": [{"from": "event.dataset", "to": "hello.world"}],
                },
            }]
        )
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        print(evt)
        print(list(evt.keys()))

        assert "dataset" not in output[0]["event"]
        assert "cpu" in output[0]["hello"]["world"]
