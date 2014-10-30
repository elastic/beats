from pbtests.packetbeat import TestCase


class Test(TestCase):

    def test_http_sample(self):
        self.render_config_template()
        self.run_packetbeat(pcap="http_minitwit.pcap")
        objs = self.read_output()

        assert len(objs) == 3
        assert all([o["type"] == "http" for o in objs])
        assert all([o["src_ip"] == "192.168.1.104" for o in objs])
        assert all([o["src_port"] == 54742 for o in objs])
        assert all([o["dst_ip"] == "192.168.1.110" for o in objs])
        assert all([o["dst_port"] == 80 for o in objs])

        assert objs[0]["status"] == "OK"
        assert objs[1]["status"] == "OK"
        assert objs[2]["status"] == "Error"
