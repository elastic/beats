from pbtests.packetbeat import TestCase


class Test(TestCase):
    def test_udpjson_config(self):
        """
        Should start with sniffer and udpjson inputs configured.
        """
        self.render_config_template(
            mysql_ports=[3306],
            input_plugins=["sniffer", "udpjson"]
        )

        self.run_packetbeat(pcap="mysql_with_whitespaces.pcap")

        objs = self.read_output()
        assert all([o["type"] == "mysql" for o in objs])
        assert len(objs) == 7

    def test_only_udpjson_config(self):
        """
        It should be possible to start without the sniffer configured.
        """
        self.render_config_template(
            input_plugins=["udpjson"]
        )

        packetbeat = self.start_packetbeat(debug_selectors=["udpjson"])

        self.wait_until(
            lambda: self.log_contains(
                msg="UDPJson plugin listening on 127.0.0.1:9712"),
            max_timeout=2)

        packetbeat.kill_and_wait()
