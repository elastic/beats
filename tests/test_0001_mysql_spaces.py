from pbtests.packetbeat import TestCase


class Test(TestCase):
    def test_mysql_with_spaces(self):
        self.render_config_template(
            mysql_ports=[3306]
        )
        self.run_packetbeat(pcap="mysql_with_whitespaces.pcap")

        objs = self.read_output()
        assert all([o["type"] == "mysql" for o in objs])
        assert len(objs) == 7
        assert all([o["port"] == 3306 for o in objs])

        assert objs[0]["mysql"]["method"] == "SET"
        assert objs[0]["mysql"]["tables"] == ""

        assert objs[2]["mysql"]["method"] == "DROP"
        assert objs[2]["mysql"]["isok"]

        assert objs[3]["mysql"]["method"] == "CREATE"
        assert objs[3]["mysql"]["isok"]

        assert objs[5]["mysql"]["method"] == "SELECT"
        assert objs[5]["mysql"]["tables"] == "test.test"
