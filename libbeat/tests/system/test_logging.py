from base import BaseTest

import re

ecs_version_log = "\"ecs.version\":"
ecs_timestamp_log = "\"@timestamp\":"
ecs_message_log = "\"message\":"
ecs_log_level_log = "\"log.level\":"


class TestLogging(BaseTest):

    def run_beat_with_args(self, msg, logging_args=[], extra_args=[]):
        self.render_config_template(
            console={"pretty": "false"}
        )
        proc = self.start_beat(logging_args=logging_args, extra_args=extra_args)
        self.wait_until(lambda: self.log_contains(msg),
                        max_timeout=2)
        proc.check_kill_and_wait()

    def assert_contains_ecs_log(self, logfile=None):
        assert self.log_contains(ecs_version_log, logfile=logfile)
        assert self.log_contains(ecs_timestamp_log, logfile=logfile)
        assert self.log_contains(ecs_message_log, logfile=logfile)
        assert self.log_contains(ecs_log_level_log, logfile=logfile)

    def test_console_ecs(self):
        """
        logs to console with ECS format
        """
        self.run_beat_with_args("mockbeat start running",
                                logging_args=["-e"])
        self.assert_contains_ecs_log()

    def test_file_default(self):
        """
        logs to file with default format
        """
        self.run_beat_with_args("Mockbeat is alive!",
                                logging_args=[])
        self.assert_contains_ecs_log(logfile="logs/mockbeat-"+self.today+".ndjson")

    def test_file_ecs(self):
        """
        logs to file with ECS format
        """
        self.run_beat_with_args("Mockbeat is alive!")
        self.assert_contains_ecs_log(logfile="logs/mockbeat-"+self.today+".ndjson")
