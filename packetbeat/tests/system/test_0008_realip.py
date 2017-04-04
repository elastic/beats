from packetbeat import BaseTest

"""
Tests for extracting the real-ip from an HTTP header.
"""


class Test(BaseTest):

    def test_x_forward_for(self):
        self.render_config_template(
            http_ports=[8002],
            http_real_ip_header="X-Forward-For",
            http_send_all_headers=True,
        )
        self.run_packetbeat(pcap="http_realip.pcap", debug_selectors=["http"])

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert o["real_ip"] == "89.247.39.104"

    def test_x_forwarded_for_multiple_ip(self):
        self.render_config_template(
            http_ports=[80],
            http_real_ip_header="X-Forwarded-For",
            http_send_all_headers=True,
        )
        self.run_packetbeat(pcap="http_x_forwarded_for.pcap", debug_selectors=["http"])

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert o["real_ip"] == "89.247.39.104"
