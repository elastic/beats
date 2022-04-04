from base import BaseTest
from beat import common_tests

import json
import os
import shutil
import signal
import subprocess
import sys
import unittest


class Test(BaseTest, common_tests.TestExportsMixin):

    def test_base(self):
        """
        Basic test with exiting Mockbeat normally
        """
        self.render_config_template(
        )

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        proc.check_kill_and_wait()
        assert self.log_contains("mockbeat stopped.")

    @unittest.skipIf(sys.platform.startswith("win"), "SIGHUP is not available on Windows")
    def test_sighup(self):
        """
        Basic test with exiting Mockbeat because of SIGHUP
        """
        self.render_config_template(
        )

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        proc.proc.send_signal(signal.SIGHUP)
        proc.check_wait()
        assert self.log_contains("mockbeat stopped.")

    def test_no_config(self):
        """
        Tests starting without a config
        """
        exit_code = self.run_beat()

        assert exit_code == 1
        assert self.log_contains("error loading config file") is True

    def test_invalid_config(self):
        """
        Checks stop on invalid config
        """
        shutil.copy(self.beat_path + "/tests/files/invalid.yml",
                    os.path.join(self.working_dir, "invalid.yml"))

        exit_code = self.run_beat(config="invalid.yml")

        assert exit_code == 1
        assert self.log_contains("error loading config file") is True

    def test_invalid_config_cli_param(self):
        """
        Checks CLI overwrite actually overwrites some config variable by
        writing an invalid value.
        """

        self.render_config_template(
            console={"pretty": "false"}
        )

        # first run with default config, validating config being
        # actually correct.
        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        proc.check_kill_and_wait()

        # start beat with invalid config setting on command line
        exit_code = self.run_beat(
            extra_args=["-d", "config", "-E", "output.console=invalid"])

        assert exit_code == 1
        assert self.log_contains("error unpacking config data") is True

    # NOTE(ph): I've removed the code to crash with theses settings, but the test is still usefull if
    # more settings are added.
    # def test_invalid_config_with_removed_settings(self):
    #     """
    #     Checks if libbeat fails to load if removed settings have been used:
    #     """
    #     self.render_config_template(console={"pretty": "false"})

    #     exit_code = self.run_beat(extra_args=[
    #         "-E", "queue_size=2048",
    #         "-E", "bulk_queue_size=1",
    #     ])

    #     assert exit_code == 1
    #     assert self.log_contains("setting 'queue_size' has been removed")
    #     assert self.log_contains("setting 'bulk_queue_size' has been removed")

    def test_console_output_timed_flush(self):
        """
        outputs/console - timed flush
        """
        self.render_config_template(
            console={"pretty": "false"}
        )

        proc = self.start_beat(logging_args=["-e"])
        self.wait_until(lambda: self.log_contains("Mockbeat is alive"),
                        max_timeout=2)
        proc.check_kill_and_wait()

    def test_console_output_size_flush(self):
        """
        outputs/console - size based flush
        """
        self.render_config_template(
            console={
                "pretty": "false",
                "bulk_max_size": 1,
            }
        )

        proc = self.start_beat(logging_args=["-e"])
        self.wait_until(lambda: self.log_contains("Mockbeat is alive"),
                        max_timeout=2)
        proc.check_kill_and_wait()

    def test_logging_metrics(self):
        self.render_config_template(
            metrics_period="0.1s"
        )
        proc = self.start_beat(logging_args=["-e"])
        self.wait_until(
            lambda: self.log_contains("Non-zero metrics in the last 100ms"),
            max_timeout=2)
        proc.check_kill_and_wait()
        self.wait_until(
            lambda: self.log_contains("Total metrics"),
            max_timeout=2)

    def test_persistent_uuid(self):
        self.render_config_template()

        # run starts and kills the beat, reading the meta file while
        # the beat is alive
        def run():
            proc = self.start_beat(extra_args=["-path.home", self.working_dir])
            self.wait_until(lambda: self.log_contains("Mockbeat is alive"),
                            max_timeout=60)

            # open meta file before killing the beat, checking the file being
            # available right after startup
            metaFile = os.path.join(self.working_dir, "data", "meta.json")
            with open(metaFile) as f:
                meta = json.loads(f.read())

            proc.check_kill_and_wait()
            return meta

        meta0 = run()
        assert self.log_contains("Beat ID: {}".format(meta0["uuid"]))

        # remove log, restart beat and check meta file did not change
        # and same UUID is used in log output.
        os.remove(os.path.join(self.working_dir, "mockbeat-" + self.today + ".ndjson"))
        meta1 = run()
        assert self.log_contains("Beat ID: {}".format(meta1["uuid"]))

        # check meta file did not change between restarts
        assert meta0 == meta1
