from packetbeat import BaseTest

"""
Tests for HTTP messages with gaps (packet loss) in them.
"""


class Test(BaseTest):

    def test_gap_in_large_file(self):
        """
        Should recover well from losing a packet in a large
        file download.
        """
        self.render_config_template(
            http_ports=[8000],
        )
        self.run_packetbeat(pcap="gap_in_stream.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert o["status"] == "OK"
        print(o["error.message"])
        assert o["error.message"] == "Packet loss while capturing the response"
