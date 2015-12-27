from metricbeat.metricbeat import BaseTest

class Test(BaseTest):

    def test_base(self):
        """
        Basic test with exiting Mockbeat normally
        """
        self.render_config_template(
        )

        proc = self.start_beat()
        self.wait_until( lambda: self.log_contains("Init Beat"))
        exit_code = proc.kill_and_wait()
        assert exit_code == 0
