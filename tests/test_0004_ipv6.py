from pbtests.packetbeat import TestCase


class Test(TestCase):

    def test_ipv6_thrift_framed(self):
        self.render_config_template(
            thrift_ports=[9090],
            thrift_transport_type="framed"
        )
        self.run_packetbeat(pcap="ipv6_thrift.pcap",
                            debug_selectors=["thrift", "ip"])

        objs = self.read_output()

        assert len(objs) == 17
        assert all([o["type"] == "thrift" for o in objs])
        assert all([o["src_ip"] == "::1" for o in objs])
        assert all([o["dst_ip"] == "::1" for o in objs])
