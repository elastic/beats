from packetbeat import BaseTest
import os
import unittest
import subprocess

"""
Tests for checking the -configtest CLI option and the
return codes.
"""


class Test(BaseTest):

    @unittest.skipIf(os.name != 'linux', "default device 'any' only exists on linux")
    def test_ok_config(self):
        """
        With 'test config' and correct configuration, it should exit with
        status 0 but not actually process any packets.
        """

        print("start test")

        self.render_config_template()

        self.run_pb_config_tst()
        assert not os.path.isfile(
            os.path.join(self.working_dir, "output/packetbeat"))

    def test_config_error(self):
        """
        With 'test config' and an error in the configuration, it should
        return a non-zero error code.
        """
        self.render_config_template(
            bpf_filter="invalid BPF filter"
        )

        self.run_pb_config_tst(exit_code=1)

    def run_pb_config_tst(self, exit_code=0):
        config = "packetbeat.yml"

        cmd = os.path.join(self.beat_path, "packetbeat.test")
        args = [
            cmd, "-systemTest",
            "-c", os.path.join(self.working_dir, config),
        ]

        if os.getenv("TEST_COVERAGE") == "true":
            args += [
                "-test.coverprofile",
                os.path.join(self.working_dir, "coverage.cov"),
            ]

        args.extend(["test", "config"])

        output = "packetbeat.log"

        with open(os.path.join(self.working_dir, output), "wb") as outfile:
            proc = subprocess.Popen(args,
                                    stdout=outfile,
                                    stderr=subprocess.STDOUT
                                    )
            actual_exit_code = proc.wait()

        if actual_exit_code != exit_code:
            print("============ Log Output =====================")
            with open(os.path.join(self.working_dir, output)) as f:
                print(f.read())
            print("============ Log End Output =====================")
        assert actual_exit_code == exit_code, "Expected exit code to be %d, but it was %d" % (
            exit_code, actual_exit_code)
        return actual_exit_code
