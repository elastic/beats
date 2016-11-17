import re
import sys
import unittest
import metricbeat
import getpass
import os

SYSTEM_CPU_FIELDS = ["idle.pct", "iowait.pct", "irq.pct", "nice.pct",
                     "softirq.pct", "steal.pct", "system.pct", "user.pct"]

SYSTEM_CPU_FIELDS_ALL = ["idle.pct", "idle.ticks", "iowait.pct", "iowait.ticks", "irq.pct", "irq.ticks", "nice.pct", "nice.ticks",
                     "softirq.pct", "softirq.ticks", "steal.pct", "steal.ticks", "system.pct", "system.ticks", "user.pct", "user.ticks"]

SYSTEM_LOAD_FIELDS = ["1", "5", "15", "norm.1", "norm.5", "norm.15"]

SYSTEM_CORE_FIELDS = ["id", "idle.pct", "iowait.pct", "irq.pct", "nice.pct",
               "softirq.pct", "steal.pct", "system.pct", "user.pct"]

SYSTEM_CORE_FIELDS_ALL = SYSTEM_CORE_FIELDS + ["idle.ticks", "iowait.ticks", "irq.ticks", "nice.ticks",
               "softirq.ticks", "steal.ticks", "system.ticks", "user.ticks"]

SYSTEM_DISKIO_FIELDS = ["name", "read.count", "write.count", "read.bytes",
                      "write.bytes", "read.time", "write.time", "io.time"]

SYSTEM_FILESYSTEM_FIELDS = ["available", "device_name", "files", "free",
                            "free_files", "mount_point", "total", "used.bytes",
                            "used.pct"]

SYSTEM_FSSTAT_FIELDS = ["count", "total_files", "total_size"]

SYSTEM_MEMORY_FIELDS = ["swap", "actual.free", "free", "total", "used.bytes", "used.pct", "actual.used.bytes",
                        "actual.used.pct"]

SYSTEM_NETWORK_FIELDS = ["name", "out.bytes", "in.bytes", "out.packets",
                         "in.packets", "in.error", "out.error", "in.dropped", "out.dropped"]

# cmdline is also part of the system process fields, but it may not be present
# for some kernel level processes. fd is also part of the system process, but
# is not available on all OSes and requires root to read for all processes.
SYSTEM_PROCESS_FIELDS = ["cpu", "memory", "name", "pid", "ppid", "pgid",
                         "state", "username"]


class SystemTest(metricbeat.BaseTest):
    @unittest.skipUnless(re.match("(?i)win|linux|darwin|freebsd|openbsd", sys.platform), "os")
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
        self.assertItemsEqual(self.de_dot(SYSTEM_CPU_FIELDS), cpu.keys())

    @unittest.skipUnless(re.match("(?i)win|linux|darwin|freebsd|openbsd", sys.platform), "os")
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
            self.assertItemsEqual(self.de_dot(SYSTEM_CPU_FIELDS_ALL), cpuStats.keys())

    @unittest.skipUnless(re.match("(?i)win|linux|darwin|freebsd|openbsd", sys.platform), "os")
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
            self.assertItemsEqual(self.de_dot(SYSTEM_CORE_FIELDS), core.keys())

    @unittest.skipUnless(re.match("(?i)win|linux|darwin|freebsd|openbsd", sys.platform), "os")
    def test_core_with_cpu_ticks(self):
        """
        Test core system output.
        """
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["core"],
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
            core = evt["system"]["core"]
            self.assertItemsEqual(self.de_dot(SYSTEM_CORE_FIELDS_ALL), core.keys())

    @unittest.skipUnless(re.match("(?i)linux|darwin|freebsd|openbsd", sys.platform), "os")
    def test_load(self):
        """
        Test system load.
        """
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["load"],
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

        cpu = evt["system"]["load"]
        self.assertItemsEqual(self.de_dot(SYSTEM_LOAD_FIELDS), cpu.keys())

    @unittest.skipUnless(re.match("(?i)win|linux|freebsd", sys.platform), "os")
    def test_diskio(self):
        """
        Test system/diskio output.
        """
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["diskio"],
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
            diskio = evt["system"]["diskio"]
            self.assertItemsEqual(self.de_dot(SYSTEM_DISKIO_FIELDS), diskio.keys())

    @unittest.skipUnless(re.match("(?i)win|linux|darwin|freebsd|openbsd", sys.platform), "os")
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
            self.assertItemsEqual(self.de_dot(SYSTEM_FILESYSTEM_FIELDS), filesystem.keys())

    @unittest.skipUnless(re.match("(?i)win|linux|darwin|freebsd|openbsd", sys.platform), "os")
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

    @unittest.skipUnless(re.match("(?i)win|linux|darwin|freebsd|openbsd", sys.platform), "os")
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
        self.assertItemsEqual(self.de_dot(SYSTEM_MEMORY_FIELDS), memory.keys())

        # Check that percentages are calculated.
        mem = memory
        if mem["total"] != 0:
            used_p = float(mem["used"]["bytes"]) / mem["total"]
            self.assertAlmostEqual(mem["used"]["pct"], used_p, places=4)

        swap = memory["swap"]
        if swap["total"] != 0:
            used_p = float(swap["used"]["bytes"]) / swap["total"]
            self.assertAlmostEqual(swap["used"]["pct"], used_p, places=4)

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
            self.assertItemsEqual(self.de_dot(SYSTEM_NETWORK_FIELDS), network.keys())

    @unittest.skipUnless(re.match("(?i)win|linux|darwin|freebsd", sys.platform), "os")
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

        found_cmdline = False
        found_fd = False
        for evt in output:
            self.assert_fields_are_documented(evt)
            process = evt["system"]["process"]
            cmdline = process.pop("cmdline", None)
            if cmdline is not None:
                found_cmdline = True
            fd = process.pop("fd", None)
            if fd is not None:
                found_fd = True
            self.assertItemsEqual(SYSTEM_PROCESS_FIELDS, process.keys())

        self.assertTrue(found_cmdline, "cmdline not found in any process events")

        if sys.platform.startswith("linux") or sys.platform.startswith("freebsd"):
            self.assertTrue(found_fd, "fd not found in any process events")

    @unittest.skipUnless(re.match("(?i)win|linux|darwin|freebsd", sys.platform), "os")
    def test_process_metricbeat(self):
        """
        Checks that the per proc stats are found in the output and
        have the expected types.
        """
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["process"],
            "period": "5s",
            "processes": ["(?i)metricbeat.test"]
        }])

        metricbeat = self.start_beat()
        self.wait_until(lambda: self.output_count(lambda x: x >= 1))
        metricbeat.check_kill_and_wait()

        output = self.read_output()[0]

        assert re.match("(?i)metricbeat.test(.exe)?", output["system.process.name"])
        assert re.match("(?i).*metricbeat.test(.exe)? -systemTest", output["system.process.cmdline"])
        assert isinstance(output["system.process.state"], basestring)
        assert isinstance(output["system.process.cpu.start_time"], basestring)
        self.check_username(output["system.process.username"])

    def check_username(self, observed, expected = None):
        if expected == None:
            expected = getpass.getuser()

        if os.name == 'nt':
            parts = observed.split("\\", 2)
            assert len(parts) == 2, "Expected proc.username to be of form DOMAIN\username, but was %s" % observed
            observed = parts[1]

        assert expected == observed, "proc.username = %s, but expected %s" % (observed, expected)
