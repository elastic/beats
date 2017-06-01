from packetbeat import BaseTest


class Test(BaseTest):
    """
    Basic Graphite Tests
    """

    def test_tds_protocol(self):
        """
        Should correctly pass for cases where the
        input follows tds protocol
        """

        self.render_config_template(
            tds_ports=[1433]
        )

        self.run_packetbeat(pcap="tds.pcap")

        objs = self.read_output()

        assert all([o["type"] == "tds" for o in objs])
        # assert all([o["bytes_in"] > 0 for o in objs])
        # assert all([o["bytes_out"] > 0 for o in objs])
        assert all([o["port"] == 1433 for o in objs])

        # assert objs[0]["request"] == ""  