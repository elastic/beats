import re
import sys
import metricbeat
import unittest


@unittest.skipUnless(re.match("(?i)win|linux|darwin|freebsd", sys.platform), "os")
class Test(metricbeat.BaseTest):

    def test_drop_fields(self):

        self.render_config_template(
            modules=[{
                "name": "system",
                "metricsets": ["cpu"],
                "period": "1s"
            }],
            processors=[{
                "drop_fields": {
                    "when": "range.system.cpu.system.pct.lt: 0.1",
                    "fields": ["system.cpu.load"],
                },
            }]
        )
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]
        self.assert_fields_are_documented(evt)

        print(evt)
        print(evt.keys())
        self.assertItemsEqual(self.de_dot([
            'agent', '@timestamp', 'system', 'metricset.module',
            'metricset.rtt', 'metricset.name', 'host', 'service', 'ecs', 'event'
        ]), evt.keys())
        cpu = evt["system"]["cpu"]
        print(cpu.keys())
        self.assertItemsEqual(self.de_dot([
            "system", "cores", "user", "softirq", "iowait",
            "idle", "irq", "steal", "nice", "total"
        ]), cpu.keys())

    def test_dropfields_with_condition(self):
        """
        Check drop_fields action works when a condition is associated.
        """
        self.render_config_template(
            modules=[{
                "name": "system",
                "metricsets": ["process"],
                "period": "1s"
            }],
            processors=[{
                "drop_fields": {
                    "fields": ["system.process.memory"],
                    "when": "range.system.process.cpu.total.pct.lt: 0.5",
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

        for event in output:
            if float(event["system.process.cpu.total.pct"]) < 0.5:
                assert "system.process.memory.size" not in event
            else:
                assert "system.process.memory.size" in event

    def test_dropevent_with_condition(self):
        """
        Check drop_event action works when a condition is associated.
        """
        self.render_config_template(
            modules=[{
                "name": "system",
                "metricsets": ["process"],
                "period": "1s"
            }],
            processors=[{
                "drop_event": {
                    "when": "range.system.process.cpu.total.pct.lt: 0.001",
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
        for event in output:
            assert float(event["system.process.cpu.total.pct"]) >= 0.001

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
                    "when.not": "contains.system.process.cmdline: metricbeat.test",
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
                "period": "1s"
            }],
            processors=[{
                "include_fields": {"fields": ["system.process.cpu", "system.process.memory"]},
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
        print(output)

        for key in [
            "system.process.cpu.start_time",
            "system.process.cpu.total.pct",
            "system.process.memory.size",
            "system.process.memory.rss.bytes",
            "system.process.memory.rss.pct"
        ]:
            assert key in output

        for key in [
            "system.process.name",
            "system.process.pid",
        ]:
            assert key not in output

    def test_multiple_actions(self):
        """
        Check the result when configuring two actions: include_fields
        and drop_fields.
        """
        self.render_config_template(
            modules=[{
                "name": "system",
                "metricsets": ["process"],
                "period": "1s"
            }],
            processors=[{
                "include_fields": {"fields": ["system.process", "process"]},
            }, {
                "drop_fields": {"fields": ["system.process.memory"]},
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
            "system.process.cpu.start_time",
            "system.process.cpu.total.pct",
            "process.name",
            "process.pid",
        ]:
            assert key in output, "'%s' not found" % key

        for key in [
            "system.process.memory.size",
            "system.process.memory.rss.bytes",
            "system.process.memory.rss.pct"
        ]:
            assert key not in output, "'%s' not expected but found" % key

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
        print(evt.keys())

        assert "dataset" not in output[0]["event"]
        assert "cpu" in output[0]["hello"]["world"]
