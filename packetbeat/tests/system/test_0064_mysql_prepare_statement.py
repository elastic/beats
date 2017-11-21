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
        assert all([o["port"] == 3307 for o in objs])
        assert len(objs) == 1

        assert objs[0]["method"] == "SELECT"
        assert objs[0]["status"] == "OK"
        assert objs[0]["params"] == "A1224638#2017/7/28 0:0:0#2017/10/28 23:59:59"
        assert objs[0]["mysql.num_rows"] == 1 

        assert all(["bytes_in" in o.keys() for o in objs])
        assert all(["bytes_out" in o.keys() for o in objs])
