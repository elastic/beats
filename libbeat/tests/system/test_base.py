from base import BaseTest

import os
import shutil
import subprocess


class Test(BaseTest):
    def test_base(self):
        """
        Basic test with exiting Mockbeat normally
        """
        self.render_config_template(
        )

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("Setup Beat"))
        proc.check_kill_and_wait()

    def test_no_config(self):
        """
        Tests starting without a config
        """
        exit_code = self.run_beat()

        assert exit_code == 1
        assert self.log_contains("error loading config file") is True
        assert self.log_contains("no such file or directory") is True

    def test_invalid_config(self):
        """
        Checks stop on invalid config
        """
        shutil.copy("../files/invalid.yml",
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
        self.wait_until(lambda: self.log_contains("Setup Beat"))
        proc.check_kill_and_wait()

        # start beat with invalid config setting on command line
        exit_code = self.run_beat(
            extra_args=["-E", "output.console=invalid"])

        assert exit_code == 1
        assert self.log_contains("error unpacking config data") is True

    def test_config_test(self):
        """
        Checks if -configtest works as expected
        """
        shutil.copy("../../_meta/config.yml",
                    os.path.join(self.working_dir, "libbeat.yml"))
        with open(self.working_dir + "/mockbeat.template.json", "w") as f:
            f.write('{"template": true}')
        with open(self.working_dir + "/mockbeat.template-es2x.json", "w") as f:
            f.write('{"template": true}')

        exit_code = self.run_beat(
            config="libbeat.yml",
            extra_args=["-configtest",
                        "-path.config", self.working_dir])

        assert exit_code == 0
        assert self.log_contains("Config OK") is True

    def test_version_simple(self):
        """
        Tests -version prints a version and exits.
        """
        self.start_beat(extra_args=["-version"]).check_wait()
        assert self.log_contains("beat version") is True

    def test_version(self):
        """
        Checks if version param works
        """
        args = ["../../libbeat.test"]

        args.extend(["-version",
                     "-e",
                     "-systemTest",
                     "-v",
                     "-d", "*",
                     "-test.coverprofile",
                     os.path.join(self.working_dir, "coverage.cov")
                     ])

        assert self.log_contains("error loading config file") is False

        with open(os.path.join(self.working_dir, "mockbeat.log"), "wb")  \
                as outputfile:
            proc = subprocess.Popen(args,
                                    stdout=outputfile,
                                    stderr=subprocess.STDOUT)
            exit_code = proc.wait()
            assert exit_code == 0

        assert self.log_contains("mockbeat") is True
        assert self.log_contains("version") is True
        assert self.log_contains("9.9.9") is True

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
                "flush_interval": "1h"
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
            lambda: self.log_contains("No non-zero metrics in the last 100ms"),
            max_timeout=2)
        proc.check_kill_and_wait()
        self.wait_until(
            lambda: self.log_contains("Total non-zero values:"),
            max_timeout=2)
