from packetbeat import BaseTest

"""
Tests that the negotiation phase at the beginning of a mysql connection
doesn't leave the parser in a broken state.
"""


class Test(BaseTest):

    def test_connection_phase(self):
        """
        This tests that requests are still captured from a mysql stream that
        starts with the "connection phase" negotiation.
        """
        self.render_config_template(
            mysql_ports=[3306],
        )
        self.run_packetbeat(pcap="mysql_connection.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        assert objs[0]['query'] == 'SELECT DATABASE()'
