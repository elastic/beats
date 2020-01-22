from packetbeat import BaseTest


class Test(BaseTest):

    def test_extended_query(self):
        self.render_config_template(
            pgsql_ports=[5432]
        )
        self.run_packetbeat(pcap="pgsql_extended_query.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]
        assert o["type"] == "pgsql"
        assert o["method"] == "SELECT"
        assert o["query"] == "SELECT * from test where id = $1"
        assert o["source.bytes"] == 90
        assert o["destination.bytes"] == 101
