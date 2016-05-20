import re
import sys
import unittest
import metricbeat

SYSTEM_CPU_FIELDS = ["idle", "iowait", "irq", "load", "nice", "softirq",
                     "steal", "system", "system_p", "user", "user_p"]

SYSTEM_CORE = ["core", "idle", "iowait", "irq", "nice", "softirq",
                "steal", "system", "system_p", "user", "user_p"]

SYSTEM_DISK_FIELDS = ["name", "read_count", "write_count", "read_bytes",
                      "write_bytes", "read_time", "write_time", "io_time"]

SYSTEM_FILESYSTEM_FIELDS = ["avail", "device_name", "files", "free",
                            "free_files", "mount_point", "total", "used",
                            "used_p"]

SYSTEM_FSSTATS_FIELDS = ["count", "total_files", "total_size"]

SYSTEM_MEMORY_FIELDS = ["swap", "mem"]

SYSTEM_PROCESS_FIELDS = ["cmdline", "cpu", "mem", "name", "pid", "ppid",
                         "state", "username"]


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
        self.assert_fields_are_documented(evt)

        cpu = evt["system"]["cpu"]
        self.assertItemsEqual(SYSTEM_CPU_FIELDS, cpu.keys())

    def test_core(self):
        """
        Test core system output.
        """
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["core"],
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()

        # Ensure no errors or warnings exist in the log.
        log = self.get_log()
        self.assertNotRegexpMatches(log, "ERR|WARN")

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            self.assert_fields_are_documented(evt)
            core = evt["system"]["core"]
            self.assertItemsEqual(SYSTEM_CORE, core.keys())

    @unittest.skipUnless(re.match("(?i)win|linux|freebsd", sys.platform), "os")
    def test_disk(self):
        """
        Test system/disk output.
        """
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["disk"],
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()

        # Ensure no errors or warnings exist in the log.
        log = self.get_log()
        self.assertNotRegexpMatches(log, "ERR|WARN")

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            self.assert_fields_are_documented(evt)
            disk = evt["system"]["disk"]
            self.assertItemsEqual(SYSTEM_DISK_FIELDS, disk.keys())

    def test_filesystem(self):
        """
        Test system/filesystem output.
        """
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["filesystem"],
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()

        # Ensure no errors or warnings exist in the log.
        log = self.get_log()
        self.assertNotRegexpMatches(log, "ERR|WARN")

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            self.assert_fields_are_documented(evt)
            filesystem = evt["system"]["filesystem"]
            self.assertItemsEqual(SYSTEM_FILESYSTEM_FIELDS, filesystem.keys())

    def test_fsstats(self):
        """
        Test system/fsstats output.
        """
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["fsstats"],
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
        self.assert_fields_are_documented(evt)

        fsstats = evt["system"]["fsstats"]
        self.assertItemsEqual(SYSTEM_FSSTATS_FIELDS, fsstats.keys())

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
        self.assert_fields_are_documented(evt)

        memory = evt["system"]["memory"]
        self.assertItemsEqual(SYSTEM_MEMORY_FIELDS, memory.keys())

        # Check that percentages are calculated.
        mem = memory["mem"]
        if mem["total"] != 0:
            used_p = float(mem["used"]) / mem["total"]
            self.assertAlmostEqual(mem["used_p"], used_p, places=4)

            used_p = float(mem["actual_used"]) / mem["total"]
            self.assertAlmostEqual(mem["actual_used_p"], used_p, places=4)

        swap = memory["swap"]
        if swap["total"] != 0:
            used_p = float(swap["used"]) / swap["total"]
            self.assertAlmostEqual(swap["used_p"], used_p, places=4)

    def test_process(self):
        """
        Test system/process output.
        """
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["process"],
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()

        # Ensure no errors or warnings exist in the log.
        log = self.get_log()
        self.assertNotRegexpMatches(log, "ERR|WARN")

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            self.assert_fields_are_documented(evt)
            process = evt["system"]["process"]
            self.assertItemsEqual(SYSTEM_PROCESS_FIELDS, process.keys())
