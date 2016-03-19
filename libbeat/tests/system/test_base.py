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
        self.wait_until( lambda: self.log_contains("Init Beat"))
        exit_code = proc.kill_and_wait()
        assert exit_code == 0

    def test_no_config(self):
        """
        Tests starting without a config
        """
        exit_code = self.run_beat()

        assert exit_code == 1
        assert self.log_contains("loading config file error") is True
        assert self.log_contains("Failed to read") is True

    def test_invalid_config(self):
        """
        Checks stop on invalid config
        """
        shutil.copy("../files/invalid.yml",
                    os.path.join(self.working_dir, "invalid.yml"))

        exit_code = self.run_beat(config="invalid.yml")

        assert exit_code == 1
        assert self.log_contains("loading config file error") is True
        assert self.log_contains("YAML config parsing failed") is True

    def test_beat_shipper_config(self):
        """
        Test that error occurs when beat and shipper are used
        """
        shutil.copy("../files/beat-shipper.yml",
                    os.path.join(self.working_dir, "beat-shipper.yml"))

        exit_code = self.run_beat(config="beat-shipper.yml")

        assert exit_code == 1
        assert self.log_contains("beat and shipper are set in config. Only one can be enabled. shipper is deprecated, use beat.") is True

    def test_shipper_to_beat(self):
        """
        Test that shipper config is copied to beat
        """
        shutil.copy("../files/shipper.yml",
                    os.path.join(self.working_dir, "shipper.yml"))

        exit_code = self.run_beat()

        proc = self.start_beat(config="shipper.yml")
        self.wait_until( lambda: self.log_contains("mockbeat sucessfully setup"))
        exit_code = proc.kill_and_wait()

        assert exit_code == 0
        assert self.log_contains("shipper config is deprecated. Please use beat instead.") is True
        assert self.log_contains("Publisher name: shipperName") is True

    def test_config_test(self):
        """
        Checks if -configtest works as expected
        """
        shutil.copy("../../etc/libbeat.yml",
                    os.path.join(self.working_dir, "libbeat.yml"))

        exit_code = self.run_beat(
                config="libbeat.yml", extra_args=["-configtest"])

        assert exit_code == 0
        assert self.log_contains("Testing configuration file") is True

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

        assert self.log_contains("loading config file error") is False

        with open(os.path.join(self.working_dir, "mockbeat.log"), "wb") \
                as outputfile:
            proc = subprocess.Popen(args,
                                    stdout=outputfile,
                                    stderr=subprocess.STDOUT)
            exit_code = proc.wait()
            assert exit_code == 0

        assert self.log_contains("mockbeat") is True
        assert self.log_contains("version") is True
        assert self.log_contains("9.9.9") is True
