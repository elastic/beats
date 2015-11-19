from pbtests.packetbeat import TestCase

"""
Tests for extracting the real-ip from an HTTP header.
"""


class Test(TestCase):

    def test_x_forward_for(self):
        self.render_config_template(
            http_ports=[8002],
            http_real_ip_header="X-Forward-For",
            http_send_all_headers=True,
            geoip_paths=["geoip_city.dat"]
        )
        self.copy_files(["geoip_city.dat"])
        self.run_packetbeat(pcap="http_realip.pcap", debug_selectors=["http"])

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert o["real_ip"] == "89.247.39.104"
        assert o["client_location"] == "52.528503, 13.410904"
