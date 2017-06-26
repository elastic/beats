import re
import sys
import unittest
from auditbeat import BaseTest


class Test(BaseTest):
    @unittest.skipUnless(re.match("(?i)linux", sys.platform), "os")
    def test_start_stop(self):
        """
        Auditbeat starts and stops without error.
        """
        self.render_config_template(modules=[{
            "name": "audit",
            "metricsets": ["kernel"],
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("start running"))
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        # Ensure all Beater stages are used.
        assert self.log_contains("Setup Beat: auditbeat")
        assert self.log_contains("auditbeat start running")
        assert self.log_contains("auditbeat stopped")
