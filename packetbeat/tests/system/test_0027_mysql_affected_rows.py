from packetbeat import BaseTest


class Test(BaseTest):

    def test_mysql_affected_rows(self):
        self.render_config_template(
            mysql_ports=[3306]
        )
        self.run_packetbeat(pcap="mysql_affected_rows.pcap",
                            debug_selectors=["mysql,tcp,publish"])

        objs = self.read_output()
        assert all([o["type"] == "mysql" for o in objs])
        assert len(objs) == 1
        assert all([o["server.port"] == 3306 for o in objs])

        assert objs[0]["method"] == "UPDATE"
        assert objs[0]["mysql.affected_rows"] == 316
        assert objs[0]["status"] == "OK"
