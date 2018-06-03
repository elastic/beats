from packetbeat import BaseTest


class Test(BaseTest):

    def test_hl7v2_accept(self):
        self.render_config_template(
            hl7v2_ports=[2575],
            hl7v2_segment_selection_mode="Include",
            hl7v2_field_selection_mode="Include",
            hl7v2_segments=[MSA],
            hl7v2_fields=[MSA-1],
        )
        self.run_packetbeat(pcap="hl7v2_application_accept.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()

        assert len(objs) == 1
        o = objs[0]
        assert o["type"] == "hl7v2"
        assert o["response-MSA-1"] == "AA"

    def test_hl7v2_reject(self):
        self.render_config_template()
        self.run_packetbeat(pcap="hl7v2_application_reject.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()

        assert len(objs) == 1
        o = objs[0]
        assert o["type"] == "hl7v2"
        assert o["response-MSA-1"] == "AR"
