import re
from metricbeat import BaseTest


class Test(BaseTest):
    def test_start_stop(self):
        """
        Metricbeat starts and stops without error.
        """
        self.render_config_template()
        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("Setup Beat"))
        proc.check_kill_and_wait()

        # Ensure no errors or warnings exist in the log.
        log = self.get_log()
        self.assertNotRegexpMatches(log, "ERR|WARN")

        # Ensure all Beater stages are used.
        self.assertRegexpMatches(log, re.compile(".*".join([
            "Setup Beat: metricbeat",
            "metricbeat start running",
            "metricbeat cleanup"
        ]), re.DOTALL))
