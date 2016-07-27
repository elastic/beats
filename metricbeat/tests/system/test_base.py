import re
import sys
import unittest
from metricbeat import BaseTest


class Test(BaseTest):
    @unittest.skipUnless(re.match("(?i)win|linux|darwin|freebsd|openbsd", sys.platform), "os")
    def test_start_stop(self):
        """
        Metricbeat starts and stops without error.
        """
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["cpu"],
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("start running"))
        proc.check_kill_and_wait()

        # Ensure no errors or warnings exist in the log.
        log = self.get_log()
        self.assertNotRegexpMatches(log, "ERR|WARN")

        # Ensure all Beater stages are used.
        assert self.log_contains("Setup Beat: metricbeat")
        assert self.log_contains("metricbeat start running")
        assert self.log_contains("metricbeat stopped")
