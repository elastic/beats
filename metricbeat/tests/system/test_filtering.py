import re
import sys
import metricbeat
import unittest

@unittest.skipUnless(re.match("(?i)win|linux|darwin|openbsd", sys.platform), "os")
class GlobalFiltering(metricbeat.BaseTest):

    def test_drop_fields(self):

        self.render_config_template(
            modules=[{
                "name": "system",
                "metricsets": ["cpu"],
                "period": "5s"
            }],
            drop_fields={
                "condition": "range.system.cpu.system_p.lt: 0.1",
                "fields": ["system.cpu.load"],
            },
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
            'beat', '@timestamp', 'system', 'metricset.module',
            'metricset.rtt', 'type', 'metricset.name'
        ]), evt.keys())
        cpu = evt["system"]["cpu"]
        print(cpu.keys())
        self.assertItemsEqual([
            "system_p", "user_p", "softirq_p", "iowait_p",
            "idle_p", "irq_p", "steal_p", "nice_p"
        ], cpu.keys())
