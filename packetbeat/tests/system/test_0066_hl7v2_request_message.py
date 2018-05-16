from packetbeat import BaseTest

"""
Tests for checking if the message id/ptient name values from HL7 v2 response messages are parsed correctly.
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
        assert o["MSH=10"] == "MSGID12349876"
        assert o["PID-5"] == "Durden^Tyler^^^Mr."

    def test_hl7v2_reject(self):
        self.render_config_template()
        self.run_packetbeat(pcap="hl7v2_application_reject.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()

        assert len(objs) == 1
        o = objs[0]
        print(o)
        assert o["type"] == "hl7v2"
        assert o["MSH=10"] == "MSGID12349877"
        assert o["PID-5"] == "Durden^^^^Mr."
