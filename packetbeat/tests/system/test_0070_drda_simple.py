from packetbeat import BaseTest

class Test(BaseTest):

    def test_simple(self):
        self.render_config_template(
            drda_ports=[1527],
        )

        self.run_packetbeat(pcap="drda/drda_simple.pcap", debug_selectors=["*"])
        objs = self.read_output()

        assert len(objs) == 21
