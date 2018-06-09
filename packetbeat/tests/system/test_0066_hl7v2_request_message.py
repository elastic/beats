from packetbeat import BaseTest


class Test(BaseTest):

    def test_hl7v2_accept(self):
        self.render_config_template()
        self.run_packetbeat(pcap="hl7v2_application_accept.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()

        assert len(objs) == 1
        o = objs[0]
        assert o["type"] == "hl7v2"
        # MSH-10 and PID-5 are fields that are configured to be selected in the test and should match the below values
        #assert o["hl7v2.request.1.MSH.10"] == "MSGID12349876"
        #assert o["hl7v2.request.2.PID.5"] == "Durden^Tyler^^^Mr."
        # MSH-11 is not a field configured to be selected in the test

    def test_hl7v2_reject(self):
        self.render_config_template()
        self.run_packetbeat(pcap="hl7v2_application_reject.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()

        assert len(objs) == 1
        o = objs[0]
        assert o["type"] == "hl7v2"
        # MSH-10 and PID-5 are fields that are configured to be selected in the test and should match the below values
        #assert o["hl7v2.request.1.MSH.10"] == "MSGID12349877"
        #assert o["hl7v2.request.2.PID.5"] == "Durden^^^^Mr."
        # MSH-11 is not a field configured to be selected in the test
