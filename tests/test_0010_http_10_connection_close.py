from pbtests.packetbeat import TestCase


class Test(TestCase):

    def test_http_sample(self):
        self.render_config_template(http_ports=['8000'])
        self.run_packetbeat(pcap="http_10_connection_close.pcap",
                            debug_selectors=["http"])
        objs = self.read_output()

        assert len(objs) == 1
        obj = objs[0]
        assert obj["status"] == "OK"
        assert obj["http"]["content_length"] == 11422
        assert obj["http"]["response"]["code"] == 200
        assert obj["type"] == "http"
        assert obj["src_ip"] == "127.0.0.1"
        assert obj["src_port"] == 37885
        assert obj["dst_ip"] == "127.0.0.1"
        assert obj["dst_port"] == 8000
