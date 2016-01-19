from winlogbeat import TestCase


"""
Contains tests for config parsing.
"""


class Test(TestCase):
    def test_valid_config(self):
        """
        With -configtest and an error in the configuration, it should
        return a non-zero error code.
        """
        self.render_config_template(
            ignore_older="1h",
            event_logs=[
              {"name": "Application", "ignore_older": "48h"}
            ]
        )
        proc = self.start_winlogbeat(extra_args=["-configtest"])
        exit_code = proc.wait()
        assert exit_code == 0

    def test_invalid_ignore_older(self):
        """
        With -configtest and an error in the configuration, it should
        return a non-zero error code.
        """
        self.render_config_template(
            ignore_older="1 hour",
            event_logs=[
              {"name": "Application"}
            ]
        )
        proc = self.start_winlogbeat(extra_args=["-configtest"])
        exit_code = proc.wait(check_exit_code=False)
        assert exit_code == 1
        assert self.log_contains("Invalid top level ignore_older value '1 hour'")
