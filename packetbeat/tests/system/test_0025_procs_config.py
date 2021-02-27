from packetbeat import BaseTest

"""
Tests for checking the procs monitoring configuration.
"""


class Test(BaseTest):

    MsgNotOnLinux = "Disabled /proc/ reading because not on linux"
    MsgEnabled = "Process watcher enabled"
    MsgDisabled = "Process watcher disabled when file input is used"

    def test_procs_default(self):
        """
        Should be disabled by default.
        """
        self.render_config_template()
        self.run_packetbeat(pcap="http_post.pcap",
                            debug_selectors=["procs"])
        objs = self.read_output()

        assert len(objs) == 1

        assert self.log_contains(self.MsgDisabled)

    def test_procs_is_enabled(self):
        """
        Should stay disabled when configured but using file input
        """
        self.render_config_template(procs_enabled=True)
        self.run_packetbeat(pcap="http_post.pcap",
                            debug_selectors=["procs"])
        objs = self.read_output()

        assert len(objs) == 1

        assert self.log_contains(self.MsgDisabled)
