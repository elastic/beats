from pbtests.packetbeat import TestCase

"""
Tests for parsing WSGI traffic.
"""


class Test(TestCase):

    def test_long_answer(self):
        self.render_config_template(
            http_ports=[8888]
        )
        self.run_packetbeat(pcap="wsgi_loopback.pcap")

        objs = self.read_output()
        assert len(objs) == 1

        o = objs[0]
        assert o["type"] == "http"
        assert o["src_port"] == 46249
        assert o["dst_port"] == 8888
        assert o["status"] == "OK"
        assert o["http"]["request"]["method"] == "GET"
        assert o["http"]["request"]["uri"] == "/"
        assert o["http"]["response"]["code"] == 200
        assert o["http"]["response"]["phrase"] == "OK"
        assert objs[0]["request_raw"] != ""
        assert objs[0]["response_raw"] != ""

    def test_drum_interraction(self):
        self.render_config_template(
            http_ports=[8888]
        )
        self.run_packetbeat(pcap="wsgi_drum.pcap")

        objs = self.read_output()
        assert len(objs) == 16

        assert all([o["type"] == "http" for o in objs])
        assert all([o["dst_port"] == 8888 for o in objs])

        assert all([o["status"] == "OK" for i, o in enumerate(objs)
            if i != 13])

        assert objs[13]["status"] == "Error"
        assert objs[13]["http"]["request"]["uri"] == "/comment/"
        assert objs[13]["http"]["response"]["code"] == 500

    def test_send_options(self):
        self.render_config_template(
            http_ports=[8888],
            http_no_send_response=True,
            http_no_send_request=True,
        )
        self.run_packetbeat(pcap="wsgi_loopback.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        assert objs[0]["request_raw"] == ""
        assert objs[0]["response_raw"] == ""
