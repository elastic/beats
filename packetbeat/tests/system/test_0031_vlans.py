from packetbeat import BaseTest

"""
Tests for traffic with VLAN tags.
"""


class Test(BaseTest):
    def test_http_vlan(self):
        """
        Should extract a request/response that have vlan tags.
        """
        self.render_config_template(
            http_ports=[8080],
        )
        self.run_packetbeat(pcap="http_over_vlan.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert o["type"] == "http"
        assert o["query"] == "GET /jpetstore/"
