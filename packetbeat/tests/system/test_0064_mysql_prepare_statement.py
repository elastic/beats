from packetbeat import BaseTest


class Test(BaseTest):

    def test_mysql_prepare_statement(self):
        self.render_config_template(
            mysql_ports=[3307]
        )
        self.run_packetbeat(pcap="mysql_prepare_statement.pcap",
                            debug_selectors=["mysql,publish"])

        objs = self.read_output()
        assert all([o["type"] == "mysql" for o in objs])
        assert all([o["server.port"] == 3307 for o in objs])
        assert len(objs) == 2

        assert objs[1]["method"] == "SELECT"
        assert objs[1]["status"] == "OK"
        assert objs[1]["params"][0] == "A1224638"
        assert objs[1]["mysql.num_rows"] == 1

        assert all(["source.bytes" in o.keys() for o in objs])
        assert all(["destination.bytes" in o.keys() for o in objs])
