from topbeat import BaseTest

import os


"""
Contains tests for system wide statistics.
"""


class Test(BaseTest):
    def test_system_wide(self):
        """
        Checks that system wide stats are found in the output and
        have the expected types.
        """
        self.render_config_template(
            system_stats=True,
            process_stats=False,
            filesystem_stats=False
        )

        topbeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=1))
        topbeat.check_kill_and_wait()
        output = self.read_output()[0]

        if os.name != "nt":
            for key in ["load1", "load5", "load15"]:
                assert type(output["load.{}".format(key)]) in [float, int]

        for key in [
            "cpu.user_p",
            "cpu.system_p",
            "mem.used_p",
            "mem.actual_used_p",
            "swap.used_p",
        ]:
            assert type(output[key]) in [float, int]

        for key in [
            "cpu.user",
            "cpu.nice",
            "cpu.system",
            "cpu.idle",
            "cpu.iowait",
            "cpu.irq",
            "cpu.softirq",
            "cpu.steal",
            "mem.total",
            "mem.used",
            "mem.free",
            "mem.actual_used",
            "mem.actual_free",
            "swap.total",
            "swap.used",
            "swap.free",
        ]:
            assert type(output[key]) is int or type(output[key]) is long
