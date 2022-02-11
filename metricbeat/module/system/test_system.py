import getpass
import metricbeat
import os
import re
import six
import sys
import unittest


SYSTEM_CPU_FIELDS_LINUX = ["cores", "idle.pct", "iowait.pct", "irq.pct", "nice.pct",
                           "softirq.pct", "steal.pct", "system.pct", "user.pct", "total.pct"]

SYSTEM_CPU_FIELDS_WINDOWS = ["cores", "idle.pct",
                             "system.pct", "user.pct", "total.pct"]

SYSTEM_CPU_FIELDS_DARWIN = ["cores", "idle.pct",
                            "system.pct", "user.pct", "total.pct", "nice.pct"]

SYSTEM_CPU_FIELDS_LINUX_ALL = ["cores", "idle.pct", "idle.ticks", "iowait.pct", "iowait.ticks", "irq.pct", "irq.ticks", "nice.pct", "nice.ticks",
                               "softirq.pct", "softirq.ticks", "steal.pct", "steal.ticks", "system.pct", "system.ticks", "user.pct", "user.ticks",
                               "idle.norm.pct", "iowait.norm.pct", "irq.norm.pct", "nice.norm.pct", "softirq.norm.pct",
                               "steal.norm.pct", "system.norm.pct", "user.norm.pct", "total.norm.pct", "total.value"]

SYSTEM_CPU_FIELDS_WINDOWS_ALL = ["cores", "idle.pct", "idle.ticks", "system.pct", "system.ticks", "user.pct", "user.ticks",
                                 "idle.norm.pct", "system.norm.pct", "user.norm.pct", "total.norm.pct", "total.value"]

SYSTEM_CPU_FIELDS_DARWIN_ALL = ["cores", "idle.pct", "idle.ticks", "nice.pct", "nice.ticks", "system.pct", "system.ticks", "user.pct", "user.ticks",
                                "idle.norm.pct", "nice.norm.pct", "system.norm.pct", "user.norm.pct", "total.norm.pct", "total.value"]

SYSTEM_LOAD_FIELDS = ["cores", "1", "5", "15", "norm.1", "norm.5", "norm.15"]

SYSTEM_CORE_FIELDS_LINUX = ["id", "idle.pct", "iowait.pct", "irq.pct", "nice.pct",
                            "softirq.pct", "steal.pct", "system.pct", "user.pct", "total.pct"]

SYSTEM_CORE_FIELDS_WINDOWS = ["id", "idle.pct",
                              "system.pct", "user.pct", "total.pct"]

SYSTEM_CORE_FIELDS_DARWIN = ["id", "idle.pct",
                             "system.pct", "user.pct", "nice.pct", "total.pct"]

SYSTEM_CORE_FIELDS_LINUX_ALL = SYSTEM_CORE_FIELDS_LINUX + ["idle.ticks", "iowait.ticks", "irq.ticks", "nice.ticks",
                                                           "softirq.ticks", "steal.ticks", "system.ticks", "user.ticks",
                                                           "idle.norm.pct", "iowait.norm.pct", "irq.norm.pct", "nice.norm.pct",
                                                           "softirq.norm.pct", "steal.norm.pct", "system.norm.pct", "user.norm.pct"]

SYSTEM_CORE_FIELDS_DARWIN_ALL = SYSTEM_CORE_FIELDS_DARWIN + ["idle.ticks", "nice.ticks", "system.ticks", "user.ticks",
                                                             "idle.norm.pct", "nice.norm.pct", "system.norm.pct", "user.norm.pct"]

SYSTEM_CORE_FIELDS_WINDOWS_ALL = SYSTEM_CORE_FIELDS_WINDOWS + ["idle.ticks", "system.ticks", "user.ticks",
                                                               "idle.norm.pct", "system.norm.pct", "user.norm.pct"]

SYSTEM_DISKIO_FIELDS = ["name", "read.count", "write.count", "read.bytes",
                        "write.bytes", "read.time", "write.time"]

SYSTEM_DISKIO_FIELDS_LINUX = ["name", "read.count", "write.count", "read.bytes",
                              "write.bytes", "read.time", "write.time", "io.time", "io.ops"]

SYSTEM_FILESYSTEM_FIELDS = ["available", "device_name", "type", "files", "free",
                            "free_files", "mount_point", "total", "used.bytes",
                            "used.pct"]

SYSTEM_FILESYSTEM_FIELDS_WINDOWS = ["available", "device_name", "type", "free",
                                    "mount_point", "total", "used.bytes",
                                    "used.pct"]

SYSTEM_FSSTAT_FIELDS = ["count", "total_files", "total_size"]

SYSTEM_MEMORY_FIELDS_LINUX = ["swap", "actual.free", "free", "total", "cached", "used.bytes", "used.pct", "actual.used.bytes",
                              "actual.used.pct"]

SYSTEM_MEMORY_FIELDS = ["swap", "actual.free", "free", "total", "used.bytes", "used.pct", "actual.used.bytes",
                                "actual.used.pct", "hugepages", "page_stats"]

SYSTEM_NETWORK_FIELDS = ["name", "out.bytes", "in.bytes", "out.packets",
                         "in.packets", "in.error", "out.error", "in.dropped", "out.dropped"]

SYSTEM_CPU_HOST_FIELDS = ["usage"]

SYSTEM_NETWORK_HOST_FIELDS = ["ingress.bytes",
                              "egress.bytes", "ingress.packets", "egress.packets"]

SYSTEM_DISK_HOST_FIELDS = ["read.bytes", "write.bytes"]

# cmdline is also part of the system process fields, but it may not be present
# for some kernel level processes. fd is also part of the system process, but
# is not available on all OSes and requires root to read for all processes.
# cgroup is only available on linux.
SYSTEM_PROCESS_FIELDS = ["cpu", "memory", "state"]


class Test(metricbeat.BaseTest):
    """
    Test Impliments the BaseTest class for the system module
    """

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
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]
        self.assert_fields_are_documented(evt)

        if "system" in evt:
            cpu = evt["system"]["cpu"]
            if sys.platform.startswith("linux"):
                self.assertCountEqual(self.de_dot(
                    SYSTEM_CPU_FIELDS_LINUX), cpu.keys())
            elif sys.platform.startswith("darwin"):
                self.assertCountEqual(self.de_dot(
                    SYSTEM_CPU_FIELDS_DARWIN), cpu.keys())
            elif sys.platform.startswith("win"):
                self.assertCountEqual(self.de_dot(
                    SYSTEM_CPU_FIELDS_WINDOWS), cpu.keys())
        else:
            host_cpu = evt["host"]["cpu"]
            self.assertCountEqual(self.de_dot(
                SYSTEM_CPU_HOST_FIELDS), host_cpu.keys())

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
                "cpu.metrics": ["percentages", "ticks"],
            },
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            self.assert_fields_are_documented(evt)
            cpuStats = evt["system"]["cpu"]
            if sys.platform.startswith("linux"):
                self.assertCountEqual(self.de_dot(
                    SYSTEM_CPU_FIELDS_LINUX_ALL), cpuStats.keys())
            elif sys.platform.startswith("win"):
                self.assertCountEqual(self.de_dot(
                    SYSTEM_CPU_FIELDS_WINDOWS_ALL), cpuStats.keys())
            elif sys.platform.startswith("darwin"):
                self.assertCountEqual(self.de_dot(
                    SYSTEM_CPU_FIELDS_DARWIN_ALL), cpuStats.keys())

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
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            self.assert_fields_are_documented(evt)
            core_stats = evt["system"]["core"]
            if sys.platform.startswith("linux"):
                self.assertCountEqual(self.de_dot(
                    SYSTEM_CORE_FIELDS_LINUX), core_stats.keys())
            elif sys.platform.startswith("win"):
                self.assertCountEqual(self.de_dot(
                    SYSTEM_CORE_FIELDS_WINDOWS), core_stats.keys())
            elif sys.platform.startswith("darwin"):
                self.assertCountEqual(self.de_dot(
                    SYSTEM_CORE_FIELDS_DARWIN), core_stats.keys())

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
                "core.metrics": ["percentages", "ticks"],
            },
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            self.assert_fields_are_documented(evt)
            core_stats = evt["system"]["core"]
            if sys.platform.startswith("linux"):
                self.assertCountEqual(self.de_dot(
                    SYSTEM_CORE_FIELDS_LINUX_ALL), core_stats.keys())
            elif sys.platform.startswith("win"):
                self.assertCountEqual(self.de_dot(
                    SYSTEM_CORE_FIELDS_WINDOWS_ALL), core_stats.keys())
            elif sys.platform.startswith("darwin"):
                self.assertCountEqual(self.de_dot(
                    SYSTEM_CORE_FIELDS_DARWIN_ALL), core_stats.keys())

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
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]
        self.assert_fields_are_documented(evt)

        cpu = evt["system"]["load"]
        self.assertCountEqual(self.de_dot(SYSTEM_LOAD_FIELDS), cpu.keys())

    @unittest.skipUnless(re.match("(?i)win|freebsd", sys.platform), "os")
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
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            self.assert_fields_are_documented(evt)
            if 'error' not in evt:
                if "system" in evt:
                    diskio = evt["system"]["diskio"]
                    self.assertCountEqual(self.de_dot(
                        SYSTEM_DISKIO_FIELDS), diskio.keys())
                elif "host" in evt:
                    host_disk = evt["host"]["disk"]
                    self.assertCountEqual(
                        SYSTEM_DISK_HOST_FIELDS, host_disk.keys())

    @unittest.skipUnless(re.match("(?i)linux", sys.platform), "os")
    def test_diskio_linux(self):
        """
        Test system/diskio output on linux.
        """
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["diskio"],
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            self.assert_fields_are_documented(evt)
            if "system" in evt:
                diskio = evt["system"]["diskio"]
                self.assertCountEqual(self.de_dot(
                    SYSTEM_DISKIO_FIELDS_LINUX), diskio.keys())
            else:
                host_disk = evt["host"]["disk"]
                self.assertCountEqual(
                    SYSTEM_DISK_HOST_FIELDS, host_disk.keys())

    @unittest.skipUnless(re.match("(?i)win|linux|darwin|freebsd|openbsd", sys.platform), "os")
    def test_filesystem(self):
        """
        Test system/filesystem output.
        """
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["filesystem"],
            "period": "5s",
            # prevent permissions issues on systems with docker
            "extras": {
                "filesystem.ignore_types": ["nsfs",
                                            "sysfs", "tmpfs", "bdev", "proc", "cgroup", "cgroup2", "cpuset",
                                            "devtmpfs", "configfs", "debugfs", "tracefs", "securityfs", "sockfs",
                                            "bpf", "pipefs", "ramfs", "hugetlbfs", "devpts", "autofs", "efivarfs",
                                            "mqueue", "selinuxfs", "binder", "pstore", "fuse", "fusectl", "rpc_pipefs",
                                            "overlay", "binfmt_misc"],
            }
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            print(evt)
            self.assert_fields_are_documented(evt)
            filesystem = evt["system"]["filesystem"]
            if sys.platform.startswith("win"):
                self.assertCountEqual(self.de_dot(
                    SYSTEM_FILESYSTEM_FIELDS_WINDOWS), filesystem.keys())
            else:
                self.assertCountEqual(self.de_dot(
                    SYSTEM_FILESYSTEM_FIELDS), filesystem.keys())

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
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]
        self.assert_fields_are_documented(evt)

        fsstat = evt["system"]["fsstat"]
        self.assertCountEqual(SYSTEM_FSSTAT_FIELDS, fsstat.keys())

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
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]
        self.assert_fields_are_documented(evt)

        memory = evt["system"]["memory"]
        if not re.match("(?i)linux", sys.platform) and not "hugepages" in memory:
            # Ensure presence of hugepages only in Linux
            memory["hugepages"] = None
        if not re.match("(?i)linux", sys.platform) and not "page_stats" in memory:
            # Ensure presence of page_stats only in Linux
            memory["page_stats"] = None

        if sys.platform.startswith("linux"):
            self.assertCountEqual(self.de_dot(
                SYSTEM_MEMORY_FIELDS_LINUX), memory.keys())
        else:
            self.assertCountEqual(self.de_dot(
                SYSTEM_MEMORY_FIELDS), memory.keys())

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
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            self.assert_fields_are_documented(evt)
            if "system" in evt:
                network = evt["system"]["network"]
                self.assertCountEqual(self.de_dot(
                    SYSTEM_NETWORK_FIELDS), network.keys())
            else:
                host_network = evt["host"]["network"]
                self.assertCountEqual(self.de_dot(
                    SYSTEM_NETWORK_HOST_FIELDS), host_network.keys())

    @unittest.skipUnless(re.match("(?i)win|linux|darwin|freebsd", sys.platform), "os")
    def test_process_summary(self):
        """
        Test system/process_summary output.
        """
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["process_summary"],
            "period": "5s",
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            self.assert_fields_are_documented(evt)

            summary = evt["system"]["process"]["summary"]
            assert isinstance(summary["total"], int)
            assert isinstance(summary["sleeping"], int)
            assert isinstance(summary["running"], int)
            assert isinstance(summary["unknown"], int)

            if not sys.platform.startswith("win"):
                assert isinstance(summary["idle"], int)
                assert isinstance(summary["stopped"], int)
                assert isinstance(summary["zombie"], int)
                assert summary["total"] == summary["sleeping"] + summary["running"] + \
                    summary["idle"] + summary["stopped"] + \
                    summary["zombie"] + summary["unknown"]

            if sys.platform.startswith("windows"):
                assert summary["total"] == summary["sleeping"] + \
                    summary["running"] + summary["unknown"]

    @unittest.skipUnless(re.match("(?i)win|linux|darwin|freebsd", sys.platform), "os")
    def test_process(self):
        """
        Test system/process output.
        """
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["process"],
            "period": "5s",
            "extras": {
                "process.env.whitelist": ["PATH"],
                "process.include_cpu_ticks": True,

                # Remove 'percpu' prior to checking documented fields because its keys are dynamic.
                "process.include_per_cpu": False,
            }
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        found_cmdline = False
        for evt in output:
            process = evt["system"]["process"]
            found_cmdline |= "cmdline" in process

            # Remove 'env' prior to checking documented fields because its keys are dynamic.
            process.pop("env", None)
            self.assert_fields_are_documented(evt)

            # Remove optional keys.
            process.pop("cgroup", None)
            process.pop("fd", None)
            process.pop("cmdline", None)

            self.assertCountEqual(SYSTEM_PROCESS_FIELDS, process.keys())

            self.assertTrue(
                found_cmdline, "cmdline not found in any process events")

    @unittest.skipUnless(re.match("(?i)linux|darwin|freebsd", sys.platform), "os")
    def test_process_unix(self):
        """
        Test system/process output for fields specific of unix systems.
        """
        import getpass

        self.render_config_template(
            modules=[{
                "name": "system",
                "metricsets": ["process"],
                "period": "5s",
                "extras": {
                    "process.env.whitelist": ["PATH"],
                    "process.include_cpu_ticks": True,

                    # Remove 'percpu' prior to checking documented fields because its keys are dynamic.
                    "process.include_per_cpu": False,
                },
            }],
            # Some info is only guaranteed in processes with permissions, check
            # only on own processes.
            processors=[{
                "drop_event": {
                    "when": "not.equals.user.name: " + getpass.getuser(),
                },
            }],
        )
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        found_fd = False
        found_env = False
        found_cwd = not sys.platform.startswith("linux")
        for evt in output:
            found_cwd |= "working_directory" in evt["process"]

            process = evt["system"]["process"]
            found_fd |= "fd" in process
            found_env |= "env" in process

            # Remove 'env' prior to checking documented fields because its keys are dynamic.
            env = process.pop("env", None)
            self.assert_fields_are_documented(evt)

            # Remove optional keys.
            process.pop("cgroup", None)
            process.pop("cmdline", None)
            process.pop("fd", None)

            self.assertCountEqual(SYSTEM_PROCESS_FIELDS, process.keys())

        if not sys.platform.startswith("darwin"):
            self.assertTrue(found_fd, "fd not found in any process events")

        self.assertTrue(found_env, "env not found in any process events")
        self.assertTrue(
            found_cwd, "working_directory not found in any process events")

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

        assert re.match("(?i)metricbeat.test(.exe)?", output["process.name"])
        assert re.match("(?i).*metricbeat.test(.exe)? -systemTest",
                        output["system.process.cmdline"])
        assert re.match("(?i).*metricbeat.test(.exe)? -systemTest",
                        output["process.command_line"])
        assert isinstance(output["system.process.state"], six.string_types)
        assert isinstance(
            output["system.process.cpu.start_time"], six.string_types)
        self.check_username(output["user.name"])

    @unittest.skipUnless(re.match("(?i)win|linux|darwin|freebsd", sys.platform), "os")
    def test_socket_summary(self):
        """
        Test system/socket_summary output.
        """
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["socket_summary"],
            "period": "5s",
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            self.assert_fields_are_documented(evt)

            summary = evt["system"]["socket"]["summary"]
            a = summary["all"]
            tcp = summary["tcp"]
            udp = summary["udp"]

            assert isinstance(a["count"], int)
            assert isinstance(a["listening"], int)

            assert isinstance(tcp["all"]["count"], int)
            assert isinstance(tcp["all"]["listening"], int)
            assert isinstance(tcp["all"]["established"], int)
            assert isinstance(tcp["all"]["close_wait"], int)
            assert isinstance(tcp["all"]["time_wait"], int)

            assert isinstance(udp["all"]["count"], int)

    @unittest.skipIf(sys.platform == "win32", "Flaky test")
    def check_username(self, observed, expected=None):
        if expected is None:
            expected = getpass.getuser()

        if os.name == 'nt':
            parts = observed.split("\\", 2)
            assert len(
                parts) == 2, "Expected proc.username to be of form DOMAIN\\username, but was %s" % observed
            observed = parts[1]

        assert expected == observed, "proc.username = %s, but expected %s" % (
            observed, expected)
