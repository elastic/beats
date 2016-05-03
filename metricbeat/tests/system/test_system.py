import re
import sys
import unittest
import metricbeat

SYSTEM_CPU_FIELDS = ["idle", "iowait", "irq", "nice", "softirq",
                     "steal", "system", "system_p", "user", "user_p"]

SYSTEM_MEMORY_FIELDS = ["swap", "mem"]


@unittest.skipUnless(re.match("(?i)win|linux|darwin|openbsd", sys.platform), "os")
class SystemTest(metricbeat.BaseTest):
    def test_cpu(self):
        """
        Test cpu system output.
        """
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["cpu"],
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()

        # Ensure no errors or warnings exist in the log.
        log = self.get_log()
        self.assertNotRegexpMatches(log, "ERR|WARN")

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        cpu = evt["system-cpu"]
        self.assertItemsEqual(SYSTEM_CPU_FIELDS, cpu.keys())

        # TODO: After fields.yml is updated this can be uncommented.
        #self.assert_fields_are_documented(evt)

    def test_memory(self):
        """
        Test system memory output.
        """
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["memory"],
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()

        # Ensure no errors or warnings exist in the log.
        log = self.get_log()
        self.assertNotRegexpMatches(log, "ERR|WARN")

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        memory = evt["system-memory"]
        self.assertItemsEqual(SYSTEM_MEMORY_FIELDS, memory.keys())

        self.assert_fields_are_documented(evt)
