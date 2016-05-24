import sys
import unittest
from topbeat import BaseTest

"""
Contains tests for CPU statistics.
"""

class Test(BaseTest):
    @unittest.skipIf(sys.platform.startswith("win"), "CPU core stats require unix")
    def test_cpu_per_core(self):
        """
        Checks that cpu usage per core statistics are exported.
        """
        self.render_config_template(
            system_stats=False,
            process_stats=False,
            filesystem_stats=False,
            core_stats=True,
        )
        topbeat = self.start_beat()
        self.wait_until(lambda: self.output_count(lambda x: x >= 1))
        topbeat.check_kill_and_wait()

        output = self.read_output()[0]

        for key in [
            "core.user_p",
            "core.system_p",
            "core.nice_p",
            "core.idle_p",
            "core.iowait_p",
            "core.irq_p",
            "core.softirq_p",
            "core.steal_p",
            "core.id",

        ]:
            assert type(output[key]) in [int, float]

    @unittest.skipIf(sys.platform.startswith("win"), "CPU core stats require unix")
    def test_cpu_per_core_with_more_details(self):
        """
        Checks that cpu usage per core statistics are exported
        when the cpu_ticks config option is enabled.
        """
        self.render_config_template(
            system_stats=False,
            process_stats=False,
            filesystem_stats=False,
            core_stats=True,
            cpu_ticks=True,
        )
        topbeat = self.start_beat()
        self.wait_until(lambda: self.output_count(lambda x: x >= 1))
        topbeat.check_kill_and_wait()

        output = self.read_output()[0]

        for key in [
            "core.user_p",
            "core.system_p",
            "core.user",
            "core.system",
            "core.nice",
            "core.nice_p",
            "core.idle",
            "core.idle_p",
            "core.iowait",
            "core.iowait_p",
            "core.irq",
            "core.irq_p",
            "core.softirq",
            "core.softirq_p",
            "core.steal",
            "core.steal_p",

        ]:
            assert type(output[key]) in [int, float]
