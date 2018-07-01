from packetbeat import BaseTest


class Test(BaseTest):

    def test_hl7_v2_accept(self):
        self.render_config_template()
        self.run_packetbeat(pcap="hl7_v2_application_accept.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()

        assert len(objs) == 1
        o = objs[0]
        assert o["type"] == "hl7"
        assert o["hl7.request.msh_message_control_id"] == "MSGID12349876"

    def test_hl7_v2_reject(self):
        self.render_config_template()
        self.run_packetbeat(pcap="hl7_v2_application_reject.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()

        assert len(objs) == 1
        o = objs[0]
        assert o["type"] == "hl7"
        assert o["hl7.request.msh_message_control_id"] == "MSGID12349877"
