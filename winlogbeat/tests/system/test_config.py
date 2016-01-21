from winlogbeat import BaseTest

"""
Contains tests for config parsing.
"""


class Test(BaseTest):

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
        self.start_beat(extra_args=["-configtest"]).check_wait()

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
        self.start_beat(extra_args=["-configtest"]).check_wait(exit_code=1)
        assert self.log_contains(
            "Invalid top level ignore_older value '1 hour'")
