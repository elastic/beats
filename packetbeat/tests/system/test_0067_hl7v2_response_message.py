from packetbeat import BaseTest

"""
Tests for checking if the acknowlegement code from HL7 v2 response messages are parsed correctly.
"""


class Test(BaseTest):

    def test_hl7v2_accept(self):
        self.render_config_template()
        self.run_packetbeat(pcap="hl7v2_application_accept.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()

        assert len(objs) == 1
        o = objs[0]
        assert o["type"] == "hl7v2"
        assert o["MSA-1"] == "AA"

    def test_hl7v2_reject(self):
        self.render_config_template()
        self.run_packetbeat(pcap="hl7v2_application_reject.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()

        assert len(objs) == 1
        o = objs[0]
        print(o)
        assert o["type"] == "hl7v2"
        assert o["MSA-1"] == "AR"
