from packetbeat import BaseTest

"""
Tests for checking expect 100-continue only generate 1 event
"""


class Test(BaseTest):

    def test_http_100_continue(self):
        """
        Should only generate one event
        """
        self.render_config_template(
            iface_device="lo0",
            http_ports=["9200"],
            http_send_all_headers=True
        )
        self.run_packetbeat(pcap="http_100_continue.pcap")
        objs = self.read_output_json()

        assert len(objs) == 1
        o = objs[0]

        assert o["type"] == "http"
        assert "request" in o["http"]
        assert "headers" in o["http"]["request"]
        assert o["http"]["request"]["headers"]["expect"] == "100-continue"

        assert "response" in o["http"]

        assert not "error" in o
