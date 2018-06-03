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
        # MSA-1 is a field that is configured to be selected in the test and should match the below value
        assert o["hl7v2.response.MSA-1"] == "AA"
        # MSH-11 is not a field configured to be selected in the test
        try:
            o["hl7v2.response.MSH-11"]
        except NameError:
            result = "ERROR"
        else:
            result = "PASS"
        assert result == "PASS"

    def test_hl7v2_reject(self):
        self.render_config_template()
        self.run_packetbeat(pcap="hl7v2_application_reject.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()

        assert len(objs) == 1
        o = objs[0]
        assert o["type"] == "hl7v2"
        # MSA-1 is a field that is configured to be selected in the test and should match the below value
        assert o["hl7v2.response.MSA-1"] == "AR"
        # MSH-11 is not a field configured to be selected in the test
        try:
            o["hl7v2.response.MSH-11"]
        except NameError:
            result = "ERROR"
        else:
            result = "PASS"
        assert result == "PASS"
