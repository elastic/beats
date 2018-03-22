from packetbeat import BaseTest


class Test(BaseTest):

    def test_2_pings(self):
        self.render_config_template()
        self.run_packetbeat(pcap="icmp/icmp_2_pings.pcap", debug_selectors=["*"])
        objs = self.read_output()

        assert len(objs) == 2
        assert all([o["icmp.version"] == 4 for o in objs])
        assert objs[0]["@timestamp"] == "2015-10-19T21:47:49.900Z"
        assert objs[0]["responsetime"] == 12
        assert objs[1]["@timestamp"] == "2015-10-19T21:47:49.924Z"
        assert objs[1]["responsetime"] == 11
        self.assert_common_fields(objs)
        self.assert_common_icmp4_fields(objs[0])
        self.assert_common_icmp4_fields(objs[1])

    def test_icmp4_ping(self):
        self.render_config_template()
        self.run_packetbeat(pcap="icmp/icmp4_ping.pcap", debug_selectors=["*"])
        objs = self.read_output()

        assert len(objs) == 1
        assert objs[0]["icmp.version"] == 4
        assert objs[0]["@timestamp"] == "2015-10-19T20:49:23.817Z"
        assert objs[0]["responsetime"] == 20
        self.assert_common_fields(objs)
        self.assert_common_icmp4_fields(objs[0])

    def test_icmp4_ping_over_vlan(self):
        self.render_config_template()
        self.run_packetbeat(pcap="icmp/icmp4_ping_over_vlan.pcap", debug_selectors=["*"])
        objs = self.read_output()

        assert len(objs) == 1
        assert objs[0]["icmp.version"] == 4
        assert objs[0]["@timestamp"] == "2015-10-19T20:49:23.849Z"
        assert objs[0]["responsetime"] == 12
        self.assert_common_fields(objs)
        self.assert_common_icmp4_fields(objs[0])

    def test_icmp6_ping(self):
        self.render_config_template()
        self.run_packetbeat(pcap="icmp/icmp6_ping.pcap", debug_selectors=["*"])
        objs = self.read_output()

        assert len(objs) == 1
        assert objs[0]["icmp.version"] == 6
        assert objs[0]["@timestamp"] == "2015-10-19T20:49:23.872Z"
        assert objs[0]["responsetime"] == 16
        self.assert_common_fields(objs)
        self.assert_common_icmp6_fields(objs[0])

    def test_icmp6_ping_over_vlan(self):
        self.render_config_template()
        self.run_packetbeat(pcap="icmp/icmp6_ping_over_vlan.pcap", debug_selectors=["*"])
        objs = self.read_output()

        assert len(objs) == 1
        assert objs[0]["icmp.version"] == 6
        assert objs[0]["@timestamp"] == "2015-10-19T20:49:23.901Z"
        assert objs[0]["responsetime"] == 12
        self.assert_common_fields(objs)
        self.assert_common_icmp6_fields(objs[0])

    def assert_common_fields(self, objs):
        assert all([o["type"] == "icmp" for o in objs])
        assert all([o["bytes_in"] == 4 for o in objs])
        assert all([o["bytes_out"] == 4 for o in objs])
        assert all([("port" in o) == False for o in objs])
        assert all([("transport" in o) == False for o in objs])

    def assert_common_icmp4_fields(self, obj):
        assert obj["ip"] == "10.0.0.2"
        assert obj["client_ip"] == "10.0.0.1"
        assert obj["path"] == "10.0.0.2"
        assert obj["status"] == "OK"
        assert obj["icmp.request.message"] == "EchoRequest(0)"
        assert obj["icmp.request.type"] == 8
        assert obj["icmp.request.code"] == 0
        assert obj["icmp.response.message"] == "EchoReply(0)"
        assert obj["icmp.response.type"] == 0
        assert obj["icmp.response.code"] == 0

    def assert_common_icmp6_fields(self, obj):
        assert obj["ip"] == "::2"
        assert obj["client_ip"] == "::1"
        assert obj["path"] == "::2"
        assert obj["status"] == "OK"
        assert obj["icmp.request.message"] == "EchoRequest(0)"
        assert obj["icmp.request.type"] == 128
        assert obj["icmp.request.code"] == 0
        assert obj["icmp.response.message"] == "EchoReply(0)"
        assert obj["icmp.response.type"] == 129
        assert obj["icmp.response.code"] == 0
