from pbtests.packetbeat import TestCase
import os

"""
Tests for checking the -test CLI option and the
return codes.
"""


class Test(TestCase):

    def test_ok_config(self):
        """
        With -test and correct configuration, it should exit with
        status 0 but not actually process any packets.
        """
        self.render_config_template()
        self.run_packetbeat(pcap="http_post.pcap", extra_args=["-configtest"])

        assert not os.path.isfile(
            os.path.join(self.working_dir, "output/packetbeat"))

    def test_config_error(self):
        """
        With -test and an error in the configuration, it should
        return a non-zero error code.
        """
        self.render_config_template(
            iface_device="NoSuchDevice"
        )
        proc = self.start_packetbeat(extra_args=["-configtest"])
        exit_code = proc.wait(check_exit_code=False)
        assert exit_code == 1
