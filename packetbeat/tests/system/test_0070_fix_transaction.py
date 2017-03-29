from packetbeat import BaseTest

"""
Tests for some basic FIX transaction types
"""


class Test(BaseTest):

    def test_ioi_message(self):
        """
        Test parsing an IOI message
        """
        self.render_config_template()
        self.run_packetbeat(pcap="fix_random_convo.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()

        # XXX Add tests here...
