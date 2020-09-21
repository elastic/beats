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
            mysql_send_request=False,
            mysql_send_response=True,
        )
        self.run_packetbeat(pcap="mysql_connection.pcap")

        objs = self.read_output()
        assert len(objs) == 5
        assert objs[0]['query'] == 'Login'
        assert objs[0]['method'] == 'LOGIN'

        assert objs[1]['query'] == 'select @@version_comment limit 1'
        assert objs[1]['method'] == 'SELECT'
        assert 'response' in objs[1]

        assert objs[3]['query'] == 'SELECT DATABASE()'
        assert objs[3]['method'] == 'SELECT'
        assert 'response' in objs[3]

        assert objs[4]['query'] == 'SHOW TABLES'
        assert objs[4]['method'] == 'SHOW'
        assert 'response' in objs[4]
