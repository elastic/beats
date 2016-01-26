from topbeat import BaseTest

import getpass
import re
import os

"""
Contains tests for per process statistics.
"""


class Test(BaseTest):
    def test_procs(self):
        """
        Checks that the per proc stats are found in the output and
        have the expected types.
        """
        self.render_config_template(
            system_stats=False,
            process_stats=True,
            filesystem_stats=False,
            proc_patterns=["(?i)topbeat.test"]  # monitor itself
        )
        topbeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=1))
        topbeat.check_kill_and_wait()

        output = self.read_output()[0]

        print output["proc.name"]
        assert re.match("(?i)topbeat.test(.exe)?", output["proc.name"])
        assert re.match("(?i).*topbeat.test(.exe)? -e -c", output["proc.cmdline"])
        assert isinstance(output["proc.state"], basestring)
        assert isinstance(output["proc.cpu.start_time"], basestring)
        self.check_username(output["proc.username"])

        for key in [
            "proc.pid",
            "proc.ppid",
            "proc.cpu.user",
            "proc.cpu.system",
            "proc.cpu.total",
            "proc.mem.size",
            "proc.mem.rss",
            "proc.mem.share",
        ]:
            assert type(output[key]) is int

        for key in [
            "proc.cpu.total_p",
            "proc.mem.rss_p",
        ]:
            assert type(output[key]) in [int, float]

    def check_username(self, observed, expected = None):
        if expected == None:
            expected = getpass.getuser()

        if os.name == 'nt':
            parts = observed.split("\\", 2)
            assert len(parts) == 2, "Expected proc.username to be of form DOMAIN\username, but was %s" % observed
            observed = parts[1]

        assert expected == observed, "proc.username = %s, but expected %s" % (observed, expected)
