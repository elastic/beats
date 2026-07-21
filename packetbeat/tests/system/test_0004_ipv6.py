from packetbeat import BaseTest


class Test(BaseTest):

    def test_ipv6_thrift_framed(self):
        self.render_config_template(
            thrift_ports=[9090],
            thrift_transport_type="framed"
        )
        pb = self.start_packetbeat(pcap="ipv6_thrift.pcap",
                                   debug_selectors=["thrift", "ip"])
        try:
            self.wait_until(lambda: self.output_lines() >= 17, max_timeout=30)
        finally:
            pb.kill_and_wait()

        objs = self.read_output()[:17]

        assert len(objs) == 17
        assert all([o["type"] == "thrift" for o in objs])
        assert all([o["client.ip"] == "::1" for o in objs])
        assert all([o["server.ip"] == "::1" for o in objs])
