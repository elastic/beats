from packetbeat import BaseTest


class Test(BaseTest):

    def test_hl7v2_accept(self):
        self.render_config_template(
            hl7v2_ports=[2575],
            hl7v2_segment_selection_mode="Include",
            hl7v2_field_selection_mode="Include",
            hl7v2_segments=[MSH, PID],
            hl7v2_fields=[MSH-10, PID-5],
        )
        self.run_packetbeat(pcap="hl7v2_application_accept.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()

        assert len(objs) == 1
        o = objs[0]
        assert o["type"] == "hl7v2"
        assert o["request-MSH-10"] == "MSGID12349876"
        assert o["request-PID-5"] == "Durden^Tyler^^^Mr."

    def test_hl7v2_reject(self):
        self.render_config_template()
        self.run_packetbeat(pcap="hl7v2_application_reject.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()

        assert len(objs) == 1
        o = objs[0]
        assert o["type"] == "hl7v2"
        assert o["request-MSH-10"] == "MSGID12349877"
        assert o["request-PID-5"] == "Durden^^^^Mr."
