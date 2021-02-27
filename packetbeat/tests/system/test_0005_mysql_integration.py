from packetbeat import BaseTest

"""
Tests based on the MySQL integration test suite.
"""


class Test(BaseTest):

    def test_string_operations(self):
        self.render_config_template(
            mysql_ports=[13001]
        )
        self.run_packetbeat(pcap="mysql_int_string_operations.pcap")

        objs = self.read_output()
        assert all([o["type"] == "mysql" for o in objs])
        assert len(objs) == 157
        assert all([o["server.port"] == 13001 for o in objs])

        assert len([o for o in objs
                    if o["method"] == "SELECT"]) == 134
        assert len([o for o in objs if o["method"] == "SHOW"]) == 10
        assert len([o for o in objs if o["method"] == "ALTER"]) == 4
        assert len([o for o in objs if o["method"] == "SET"]) == 3
        assert len([o for o in objs if o["method"] == "CREATE"]) == 2
        assert len([o for o in objs if o["method"] == "CREATE"]) == 2
