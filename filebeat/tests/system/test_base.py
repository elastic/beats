from filebeat import TestCase

import os
import shutil
import subprocess



class Test(TestCase):
    def test_base(self):
        """
        Basic test with exiting filebeat normally
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*"
        )

        filebeat = self.start_filebeat()
        self.wait_until(lambda: self.log_contains("Start running"))

        exit_code = filebeat.kill_and_wait()
        assert exit_code == 0, "Exit code was %d" % exit_code


    def test_no_config(self):
        """
        Tests starting without a config
        """
        exit_code = self.run_filebeat(check_exit_code=False)

        assert exit_code == 1
        assert True == self.log_contains("loading config file error")
        assert True == self.log_contains("Failed to read")


    def test_invalid_config(self):
        """
        Checks stop on invalid config
        """
        shutil.copy("../files/invalid.yml", os.path.join(self.working_dir, "invalid.yml"))

        exit_code = self.run_filebeat(config="invalid.yml", check_exit_code=False)

        assert exit_code == 1
        assert True == self.log_contains("loading config file error")
        assert True == self.log_contains("YAML config parsing failed")


    def test_config_test(self):
        """
        Checks if -configtest works as expected
        """
        shutil.copy("../../etc/filebeat.yml", os.path.join(self.working_dir, "filebeat.yml"))

        self.run_filebeat(config="filebeat.yml", extra_args=["-configtest"])

        assert True == self.log_contains("Testing configuration file")

    def test_version(self):
        """
        Checks if version param works
        """
        args = ["../../filebeat.test"]

        args.extend(["-version",
                     "-e",
                     "-systemTest",
                     "-v",
                     "-d", "*",
                     "-test.coverprofile",
                     os.path.join(self.working_dir, "coverage.cov")
                     ])

        assert False == self.log_contains("loading config file error")

        with open(os.path.join(self.working_dir, "filebeat.log"), "wb") as outputfile:
            proc = subprocess.Popen(args,
                                    stdout=outputfile,
                                    stderr=subprocess.STDOUT)
            exit_code = proc.wait()
            assert exit_code == 0, "Exit code was %d" % exit_code

        assert True == self.log_contains("filebeat")
        assert True == self.log_contains("version")



