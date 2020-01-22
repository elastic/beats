from packetbeat import BaseTest

"""
Tests for MySQL messages with Windows line ending \\r\\n.
"""


class Test(BaseTest):

    def test_rn(self):
        """
        Should get method "SELECT" instead of "SELECT\\r"
        """
        self.render_config_template(
            mysql_ports=[3306],
        )
        self.run_packetbeat(pcap="mysql_windows_lineending.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert o["status"] == "OK"
        assert o["method"] == "SELECT"
