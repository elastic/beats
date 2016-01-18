from mockbeat import TestCase

import os
import shutil
import subprocess


# Additional tests to be added:
# * Check what happens when file renamed -> no recrawling should happen
# * Check if file descriptor is "closed" when file disappears
class Test(TestCase):
    def test_base(self):
        """
        Basic test with exiting Mockbeat normally
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*"
        )

        exit_code = self.run_mockbeat()
        assert exit_code == 0


    def test_no_config(self):
        """
        Tests starting without a config
        """
        exit_code = self.run_mockbeat()

        assert exit_code == 1
        assert True == self.log_contains("loading config file error")
        assert True == self.log_contains("Failed to read")


    def test_invalid_config(self):
        """
        Checks stop on invalid config
        """
        shutil.copy("../files/invalid.yml", os.path.join(self.working_dir, "invalid.yml"))

        exit_code = self.run_mockbeat(config="invalid.yml")

        assert exit_code == 1
        assert True == self.log_contains("loading config file error")
        assert True == self.log_contains("YAML config parsing failed")


    def test_config_test(self):
        """
        Checks if -configtest works as expected
        """
        shutil.copy("../../etc/libbeat.yml", os.path.join(self.working_dir, "libbeat.yml"))

        exit_code = self.run_mockbeat(config="libbeat.yml", extra_args=["-configtest"])

        assert exit_code == 0
        assert True == self.log_contains("Testing configuration file")

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

        assert False == self.log_contains("loading config file error")

        with open(os.path.join(self.working_dir, "mockbeat.log"), "wb") as outputfile:
            proc = subprocess.Popen(args,
                                    stdout=outputfile,
                                    stderr=subprocess.STDOUT)
            exit_code = proc.wait()
            assert exit_code == 0

        assert True == self.log_contains("mockbeat")
        assert True == self.log_contains("version")
        assert True == self.log_contains("9.9.9")
