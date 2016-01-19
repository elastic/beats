from packetbeat import BaseTest

"""
Tests for checking the procs monitoring configuration.
"""


class Test(BaseTest):

    MsgNotOnLinux = "Disabled /proc/ reading because not on linux"
    MsgEnabled = "Process matching enabled"

    def test_procs_default(self):
        """
        Should be disabled by default.
        """
        self.render_config_template()
        self.run_packetbeat(pcap="http_post.pcap",
                            debug_selectors=["procs"])
        objs = self.read_output()

        assert len(objs) == 1

        assert not self.log_contains(self.MsgNotOnLinux) and \
            not self.log_contains(self.MsgEnabled)

    def test_procs_is_enabled(self):
        """
        Should be enabled when configured (but can be still
        disabled if we're not on Linux.
        """
        self.render_config_template(procs_enabled=True)
        self.run_packetbeat(pcap="http_post.pcap",
                            debug_selectors=["procs"])
        objs = self.read_output()

        assert len(objs) == 1

        assert self.log_contains(self.MsgNotOnLinux) or \
            self.log_contains(self.MsgEnabled)
