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
        Checks that cpu usage per core statistics are exported
        when the config option is enabled.
        """
        self.render_config_template(
            system_stats=True,
            process_stats=False,
            filesystem_stats=False,
            cpu_per_core=True
        )
        topbeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=1))
        topbeat.check_kill_and_wait()

        output = self.read_output()[0]

        for key in [
            "cpus.cpu0.user_p",
            "cpus.cpu0.system_p",
            "cpus.cpu0.user",
            "cpus.cpu0.system",
            "cpus.cpu0.nice",
            "cpus.cpu0.idle",
            "cpus.cpu0.iowait",
            "cpus.cpu0.irq",
            "cpus.cpu0.softirq",
            "cpus.cpu0.steal",

        ]:
            assert type(output[key]) in [int, float]
