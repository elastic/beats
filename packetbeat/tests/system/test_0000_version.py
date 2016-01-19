from pbtests.packetbeat import TestCase

import os
import shutil
import subprocess



class Test(TestCase):
    def test_version(self):
        """
        Checks if version param works
        """
        args = ["../../packetbeat.test"]

        args.extend(["-version",
                     "-e",
                     "-systemTest",
                     "-v",
                     "-d", "*",
                     "-test.coverprofile",
                     os.path.join(self.working_dir, "coverage.cov")
                     ])

        assert False == self.log_contains("loading config file error")

        with open(os.path.join(self.working_dir, "packetbeat.log"), "wb") as outputfile:
            proc = subprocess.Popen(args,
                                    stdout=outputfile,
                                    stderr=subprocess.STDOUT)
            exit_code = proc.wait()
            assert exit_code == 0, "Exit code was %d" % exit_code

        assert True == self.log_contains("packetbeat")
        assert True == self.log_contains("version")


