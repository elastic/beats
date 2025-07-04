"""
Metricbeat system module tests
"""
import getpass
import os
import platform
import re
import sys
import unittest
import six
import metricbeat  # pylint: disable=import-error

SYSTEM_CPU = {
    metricbeat.P_WIN: ["cores", "idle.pct",
                       "system.pct", "user.pct", "total.pct"]
}
SYSTEM_CPU[metricbeat.P_DARWIN] = SYSTEM_CPU[metricbeat.P_WIN] + ["nice.pct"]
SYSTEM_CPU[metricbeat.P_LINUX] = SYSTEM_CPU[metricbeat.P_DARWIN] + ["iowait.pct", "irq.pct", "nice.pct",
                                                                    "softirq.pct", "steal.pct"]
SYSTEM_CPU_HOST_FIELDS = ["usage"]


SYSTEM_CPU_ALL = {
    metricbeat.P_WIN: SYSTEM_CPU[metricbeat.P_WIN] + ["idle.ticks", "system.ticks", "user.ticks",
                                                      "idle.norm.pct", "system.norm.pct",
                                                      "user.norm.pct", "total.norm.pct", "total.value"]
}
SYSTEM_CPU_ALL[metricbeat.P_DARWIN] = SYSTEM_CPU[metricbeat.P_DARWIN] + \
    ["nice.ticks", "nice.norm.pct"]

SYSTEM_CPU_ALL[metricbeat.P_LINUX] = SYSTEM_CPU[metricbeat.P_LINUX] + ["idle.ticks", "iowait.ticks",
                                                                       "irq.ticks", "softirq.ticks", "nice.ticks",
                                                                       "steal.ticks", "system.ticks", "user.ticks",
                                                                       "idle.norm.pct", "iowait.norm.pct",
                                                                       "irq.norm.pct", "nice.norm.pct",
                                                                       "softirq.norm.pct", "steal.norm.pct",
                                                                       "system.norm.pct", "user.norm.pct",
                                                                       "total.norm.pct", "total.value"]

# metrics from /proc/cpuinfo are platform dependent and only
# reliably available on Linux x86-like platforms


def is_cpuinfo_supported():
    return platform.machine() in {'i386', 'i686', 'x86_64', 'amd64'}


SYSTEM_CORE_CPUINFO_FIELDS = ["model_name", "model_number", "mhz",
                              "core_id", "physical_id"]

SYSTEM_CORE = {
    metricbeat.P_WIN: ["id", "idle.pct",
                       "system.pct", "user.pct", "total.pct"]
}
SYSTEM_CORE[metricbeat.P_DARWIN] = SYSTEM_CORE[metricbeat.P_WIN] + ["nice.pct"]
SYSTEM_CORE[metricbeat.P_LINUX] = SYSTEM_CORE[metricbeat.P_DARWIN] + \
    ["iowait.pct", "irq.pct", "softirq.pct", "steal.pct"]

if is_cpuinfo_supported():
    SYSTEM_CORE[metricbeat.P_LINUX].extend(SYSTEM_CORE_CPUINFO_FIELDS)

SYSTEM_CORE_ALL = {
    metricbeat.P_WIN: SYSTEM_CORE[metricbeat.P_WIN] + ["idle.ticks", "system.ticks", "user.ticks",
                                                       "idle.norm.pct", "system.norm.pct", "user.norm.pct"]
}
SYSTEM_CORE_ALL[metricbeat.P_DARWIN] = SYSTEM_CORE[metricbeat.P_DARWIN] + ["idle.ticks", "nice.ticks",
                                                                           "system.ticks", "user.ticks",
                                                                           "idle.norm.pct", "nice.norm.pct",
                                                                           "system.norm.pct", "user.norm.pct"]

SYSTEM_CORE_ALL[metricbeat.P_LINUX] = SYSTEM_CORE[metricbeat.P_LINUX] + ["idle.ticks", "iowait.ticks",
                                                                         "irq.ticks", "nice.ticks",
                                                                         "softirq.ticks", "steal.ticks",
                                                                         "system.ticks", "user.ticks",
                                                                         "idle.norm.pct", "iowait.norm.pct",
                                                                         "irq.norm.pct", "nice.norm.pct",
                                                                         "softirq.norm.pct", "steal.norm.pct",
                                                                         "system.norm.pct", "user.norm.pct"]

SYSTEM_LOAD_FIELDS = ["cores", "1", "5", "15", "norm.1", "norm.5", "norm.15"]

SYSTEM_DISKIO = {
    metricbeat.P_DEF: ["name", "read.count", "write.count", "read.bytes",
                       "write.bytes", "read.time", "write.time"]
}
SYSTEM_DISKIO[metricbeat.P_LINUX] = SYSTEM_DISKIO[metricbeat.P_DEF] + \
    ["io.time", "io.ops"]

SYSTEM_FILESYSTEM = {
    metricbeat.P_WIN: ["available", "device_name", "type", "free",
                                    "mount_point", "total", "used.bytes",
                                    "used.pct"]
}
SYSTEM_FILESYSTEM[metricbeat.P_DEF] = SYSTEM_FILESYSTEM[metricbeat.P_WIN] + \
    ["files", "free_files", "options"]


SYSTEM_FSSTAT_FIELDS = ["count", "total_files", "total_size"]

SYSTEM_MEMORY = {
    metricbeat.P_DEF: ["swap", "actual.free", "free", "total", "used.bytes", "used.pct", "actual.used.bytes",
                       "actual.used.pct"]
}
SYSTEM_MEMORY[metricbeat.P_LINUX] = SYSTEM_MEMORY[metricbeat.P_DEF] + \
    ["cached", "hugepages", "page_stats"]

SYSTEM_MEMORY_FIELDS_LINUX = ["swap", "actual.free", "free", "total", "cached", "used.bytes", "used.pct", "actual.used.bytes",
                              "actual.used.pct"]

SYSTEM_MEMORY_FIELDS = ["swap", "actual.free", "free", "total", "used.bytes", "used.pct", "actual.used.bytes",
                                "actual.used.pct", "hugepages", "page_stats"]

SYSTEM_NETWORK_FIELDS = ["name", "out.bytes", "in.bytes", "out.packets",
                         "in.packets", "in.error", "out.error", "in.dropped", "out.dropped"]


SYSTEM_NETWORK_HOST_FIELDS = ["ingress.bytes",
                              "egress.bytes", "ingress.packets", "egress.packets"]

SYSTEM_DISK_HOST_FIELDS = ["read.bytes", "write.bytes"]


# cmdline is also part of the system process fields, but it may not be present
# for some kernel level processes. fd is also part of the system process, but
# is not available on all OSes and requires root to read for all processes.
# num_threads may not be readable for some privileged process on Windows,
# cgroup is only available on linux.
SYSTEM_PROCESS_FIELDS = ["cpu", "memory", "state", "io"]


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
        self.run_beat_and_stop()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]
        self.assert_fields_are_documented(evt)

        if "system" in evt:
            cpu = evt["system"]["cpu"]
            self.assert_fields_for_platform(SYSTEM_CPU, cpu)
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
        self.run_beat_and_stop()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            self.assert_fields_are_documented(evt)
            cpu_stats = evt["system"]["cpu"]
            self.assert_fields_for_platform(SYSTEM_CPU_ALL, cpu_stats)

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
        self.run_beat_and_stop()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            self.assert_fields_are_documented(evt)
            core_stats = evt["system"]["core"]
            if sys.platform == metricbeat.P_LINUX and not is_cpuinfo_supported():
                for f in SYSTEM_CORE_CPUINFO_FIELDS:
                    core_stats.pop(f, None)
            self.assert_fields_for_platform(SYSTEM_CORE, core_stats)

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
        self.run_beat_and_stop()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            self.assert_fields_are_documented(evt)
            core_stats = evt["system"]["core"]
            if sys.platform == metricbeat.P_LINUX and not is_cpuinfo_supported():
                for f in SYSTEM_CORE_CPUINFO_FIELDS:
                    core_stats.pop(f, None)
            self.assert_fields_for_platform(SYSTEM_CORE_ALL, core_stats)

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
        self.run_beat_and_stop()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]
        self.assert_fields_are_documented(evt)

        cpu = evt["system"]["load"]
        self.assertCountEqual(self.de_dot(SYSTEM_LOAD_FIELDS), cpu.keys())

    @unittest.skipUnless(re.match("(?i)linux|freebsd|openbsd|win", sys.platform), "os")
    def test_diskio(self):
        """
        Test system/diskio output.
        """
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["diskio"],
            "period": "5s"
        }])
        self.run_beat_and_stop()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            self.assert_fields_are_documented(evt)
            if 'error' not in evt:
                if "system" in evt:
                    diskio = evt["system"]["diskio"]
                    self.remove_fields(diskio, ["serial_number"])
                    self.assert_fields_for_platform(SYSTEM_DISKIO, diskio)
                elif "host" in evt:
                    host_disk = evt["host"]["disk"]
                    self.assertCountEqual(
                        SYSTEM_DISK_HOST_FIELDS, host_disk.keys())

    @unittest.skipUnless(re.match("(?i)win|linux|freebsd|openbsd", sys.platform), "os")
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
                                            "overlay", "binfmt_misc", "unavailable"],
            }
        }])
        self.run_beat_and_stop()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            self.assert_fields_are_documented(evt)
            filesystem = evt["system"]["filesystem"]
            self.assert_fields_for_platform(SYSTEM_FILESYSTEM, filesystem)

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
        self.run_beat_and_stop()

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
        self.run_beat_and_stop()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]
        self.assert_fields_are_documented(evt)

        memory = evt["system"]["memory"]
        # these fields may not be on the event depending on the host system
        if re.match("(?i)linux", sys.platform) and not "hugepages" in memory:
            # Ensure presence of hugepages only in Linux
            memory["hugepages"] = None
        if re.match("(?i)linux", sys.platform) and not "page_stats" in memory:
            # Ensure presence of page_stats only in Linux
            memory["page_stats"] = None

        self.assert_fields_for_platform(SYSTEM_MEMORY, memory)

        # Check that percentages are calculated.
        if memory["total"] != 0:
            used_p = float(memory["used"]["bytes"]) / memory["total"]
            self.assertAlmostEqual(memory["used"]["pct"], used_p, places=4)

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
        self.run_beat_and_stop()

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
        self.run_beat_and_stop()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)
        only_errors_encountered = True

        for evt in output:
            self.assert_fields_are_documented(evt)
            if evt.get("error", None) is not None:
                # Here, we assume that the error is non-fatal and we move forward the test execution.
                # If the error is non-fatal, the test should pass with assertions.
                # If the error is fatal, the test should fail.
                continue

            # we've encoutered an event. Turn off the flag
            only_errors_encountered = False

            summary = evt["system"]["process"]["summary"]
            assert isinstance(summary["total"], int)

            if sys.platform.startswith("windows"):
                assert isinstance(summary["running"], int)
                assert isinstance(summary["total"], int)

        # If the flag is true, we've only encountered error (fatal errors)
        # If the flag is false, we've encoutered events and probably some non-fatal errors.
        assert not only_errors_encountered

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
        self.run_beat_and_stop()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        found_cmdline = False
        only_errors_encountered = True
        for evt in output:
            if evt.get("error", None) is not None:
                # Here, we assume that the error is non-fatal and we move forward the test execution.
                # If the error is non-fatal, the test should pass with assertions.
                # If the error is fatal, the test should fail.
                continue

            # we've encoutered an event. Turn off the flag
            only_errors_encountered = False

            process = evt["system"]["process"]
            # Not all process will have 'cmdline' due to permission issues,
            # especially on Windows. Therefore we ensure at least some of
            # them will have it.
            found_cmdline |= "cmdline" in process

            # Remove 'env' prior to checking documented fields because its keys are dynamic.
            process.pop("env", None)
            self.assert_fields_are_documented(evt)

            # Remove optional keys.
            process.pop("cgroup", None)
            process.pop("fd", None)
            process.pop("cmdline", None)
            process.pop("num_threads", None)

            self.assertCountEqual(SYSTEM_PROCESS_FIELDS, process.keys())
        # After iterating over all process, make sure at least one of them had
        # the 'cmdline' set.
        self.assertTrue(
            found_cmdline, "cmdline not found in any process events")

        # If the flag is true, we've only encountered error (fatal errors)
        # If the flag is false, we've encoutered events and probably some non-fatal errors.
        assert not only_errors_encountered

    @unittest.skipUnless(re.match("(?i)linux|darwin|freebsd", sys.platform), "os")
    def test_process_unix(self):
        """
        Test system/process output checking it has got all expected fields specific of unix systems and no extra ones.
        """

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
        self.run_beat_and_stop()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        found_fd = False
        found_env = False
        found_cwd = not sys.platform.startswith("linux")
        only_errors_encountered = True
        for evt in output:
            if evt.get("error", None) is not None:
                # Here, we assume that the error is non-fatal and we move forward the test execution.
                # If the error is non-fatal, the test should pass with assertions.
                # If the error is fatal, the test should fail.
                continue
            # we've encoutered an event. Turn off the flag
            only_errors_encountered = False

            found_cwd |= "working_directory" in evt["process"]

            process = evt["system"]["process"]
            found_fd |= "fd" in process
            found_env |= "env" in process

            # Remove 'env' prior to checking documented fields because its keys are dynamic.
            process.pop("env", None)
            self.assert_fields_are_documented(evt)

            # Remove optional keys.
            process.pop("cgroup", None)
            process.pop("cmdline", None)
            process.pop("fd", None)
            process.pop("num_threads", None)

            self.assertCountEqual(SYSTEM_PROCESS_FIELDS, process.keys())

        if not sys.platform.startswith("darwin"):
            self.assertTrue(found_fd, "fd not found in any process events")

        # If the flag is true, we've only encountered error (fatal errors)
        # If the flag is false, we've encoutered events and probably some non-fatal errors.
        assert not only_errors_encountered
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

        mb_handle = self.start_beat()
        self.wait_until(lambda: self.output_count(lambda x: x >= 1))
        mb_handle.check_kill_and_wait()

        output = self.read_output()[0]

        assert re.match("(?i)metricbeat.test(.exe)?", output["process.name"])
        assert re.match("(?i).*metricbeat.test(.exe)? --systemTest",
                        output["system.process.cmdline"])
        assert re.match("(?i).*metricbeat.test(.exe)? --systemTest",
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
        self.run_beat_and_stop()

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
        """
        check username value
        """
        if expected is None:
            expected = getpass.getuser()

        if os.name == 'nt':
            parts = observed.split("\\", 2)
            assert len(
                parts) == 2, f"Expected proc.username to be of form DOMAIN\\username, but was {observed}"
            observed = parts[1]

        assert expected == observed, f"proc.username = {observed}, but expected {expected}"
