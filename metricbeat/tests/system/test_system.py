import re
import sys
import unittest
import metricbeat

SYSTEM_CPU_FIELDS = ["idle_p", "iowait_p", "irq_p", "load", "nice_p",
                     "softirq_p", "steal_p", "system_p", "user_p"]

SYSTEM_CPU_ALL_FIELDS = ["idle_p", "idle", "iowait_p", "iowait", "irq_p", "irq", "load", "nice_p", "nice",
                     "softirq_p", "softirq", "steal_p", "steal", "system_p", "system", "user_p", "user"]

SYSTEM_CORE = ["id", "idle_p", "iowait_p", "irq_p", "nice_p",
               "softirq_p", "steal_p", "system_p", "user_p"]

SYSTEM_DISK_FIELDS = ["name", "read_count", "write_count", "read_bytes",
                      "write_bytes", "read_time", "write_time", "io_time"]

SYSTEM_FILESYSTEM_FIELDS = ["avail", "device_name", "files", "free",
                            "free_files", "mount_point", "total", "used",
                            "used_p"]

SYSTEM_FSSTAT_FIELDS = ["count", "total_files", "total_size"]

SYSTEM_MEMORY_FIELDS = ["swap", "mem"]

SYSTEM_NETWORK_FIELDS = ["name", "bytes_sent", "bytes_recv", "packets_sent",
                         "packets_recv", "errin", "errout", "dropin", "dropout"]

SYSTEM_PROCESS_FIELDS = ["cmdline", "cpu", "mem", "name", "pid", "ppid",
                         "state", "username"]


class SystemTest(metricbeat.BaseTest):
    @unittest.skipUnless(re.match("(?i)win|linux|darwin|openbsd", sys.platform), "os")
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

    @unittest.skipUnless(re.match("(?i)win|linux|darwin|openbsd", sys.platform), "os")
    def test_cpu_ticks_option(self):
        """
        Test cpu_ticks configuration option.
        """
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["cpu"],
            "period": "5s",
            "extras": {
                "cpu_ticks": True,
            },
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
            cpuStats = evt["system"]["cpu"]
            self.assertItemsEqual(SYSTEM_CPU_ALL_FIELDS, cpuStats.keys())

    @unittest.skipUnless(re.match("(?i)linux|darwin|openbsd", sys.platform), "os")
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

    @unittest.skipUnless(re.match("(?i)win|linux|darwin|openbsd", sys.platform), "os")
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

    @unittest.skipUnless(re.match("(?i)win|linux|darwin|openbsd", sys.platform), "os")
    def test_fsstat(self):
        """
        Test system/fsstat output.
        """
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["fsstat"],
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

        fsstat = evt["system"]["fsstat"]
        self.assertItemsEqual(SYSTEM_FSSTAT_FIELDS, fsstat.keys())

    @unittest.skipUnless(re.match("(?i)win|linux|darwin|openbsd", sys.platform), "os")
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

    @unittest.skipUnless(re.match("(?i)darwin|win|linux|freebsd", sys.platform), "os")
    def test_network(self):
        """
        Test system/network output.
        """
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["network"],
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
            network = evt["system"]["network"]
            self.assertItemsEqual(SYSTEM_NETWORK_FIELDS, network.keys())

    @unittest.skipUnless(re.match("(?i)win|linux|darwin", sys.platform), "os")
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
