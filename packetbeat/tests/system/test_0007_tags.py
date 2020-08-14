from packetbeat import BaseTest

"""
Tests for tags handling.
"""


class Test(BaseTest):

    def test_tags(self):
        """
        Configured tags should show up as an array.
        """
        self.render_config_template(
            http_ports=[8888],
            agent_tags=["nginx", "wsgi", "drum"]
        )
        self.run_packetbeat(pcap="wsgi_loopback.pcap")

        objs = self.read_output()
        assert len(objs) == 1

        o = objs[0]
        assert "tags" in o
        assert o["tags"] == ["nginx", "wsgi", "drum"]

    def test_empty_tags(self):
        """
        If no tags are defined, the field can be
        missing.
        """
        self.render_config_template(
            http_ports=[8888],
        )
        self.run_packetbeat(pcap="wsgi_loopback.pcap")

        objs = self.read_output()
        assert len(objs) == 1

        o = objs[0]
        assert "tags" not in o
