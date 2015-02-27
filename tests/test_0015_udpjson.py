from pbtests.packetbeat import TestCase


class Test(TestCase):
    def test_udpjson_config(self):
        """
        Should start with udpjson inputs configured.
        """
        self.render_config_template(
            mysql_ports=[3306],
            inputs=["sniffer", "udpjson"]
        )

        self.run_packetbeat(pcap="mysql_with_whitespaces.pcap")

        objs = self.read_output()
        assert all([o["type"] == "mysql" for o in objs])
        assert len(objs) == 7
