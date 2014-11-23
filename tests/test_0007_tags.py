from pbtests.packetbeat import TestCase

"""
Tests for parsing WSGI traffic.
"""


class Test(TestCase):

    def test_long_answer(self):
        self.render_config_template(
            http_ports=[8888],
            agent_tags=["nginx", "wsgi", "drum"]
        )
        self.run_packetbeat(pcap="wsgi_loopback.pcap")

        objs = self.read_output()
        assert len(objs) == 1

        o = objs[0]
        assert o["tags"] == "nginx wsgi drum"
