from packetbeat import BaseTest


class Test(BaseTest):

    def test_mysql_error(self):
        self.render_config_template(
            mysql_ports=[3306]
        )
        self.run_packetbeat(pcap="mysql_err_database_not_selected.pcap",
                            debug_selectors=["mysql,tcp,publish"])

        objs = self.read_output()
        assert all([o["type"] == "mysql" for o in objs])
        assert len(objs) == 1
        assert all([o["port"] == 3306 for o in objs])

        assert objs[0]["method"] == "SELECT"
        assert objs[0]["status"] == "Error"
        assert objs[0]["mysql.error_code"] == 1046
        assert objs[0]["mysql.error_message"] == "3D000: No database selected"
