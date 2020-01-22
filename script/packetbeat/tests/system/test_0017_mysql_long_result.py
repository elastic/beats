from packetbeat import BaseTest

"""
Tests for trimming long results in mysql.
"""


class Test(BaseTest):

    def test_default_settings(self):
        """
        Should store the entire rows but only
        10 rows with default settings.
        """
        self.render_config_template(
            mysql_ports=[3306],
            mysql_send_response=True
        )
        self.run_packetbeat(pcap="mysql_long_result.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        res = objs[0]
        assert res["mysql.num_rows"] == 15

        lines = res["response"].strip().split("\n")
        assert len(lines) == 11    # 10 plus header

        for line in lines[3:]:
            print(len(line))
            assert len(line) == 261

    def test_max_row_length(self):
        """
        Should be able to cap the row length.
        """
        self.render_config_template(
            mysql_ports=[3306],
            mysql_max_row_length=79,
            mysql_send_response=True
        )
        self.run_packetbeat(pcap="mysql_long_result.pcap",
                            debug_selectors=["mysqldetailed"])

        objs = self.read_output()
        assert len(objs) == 1
        res = objs[0]
        assert res["mysql.num_rows"] == 15

        lines = res["response"].strip().split("\n")
        assert len(lines) == 11    # 10 plus header

        for line in lines[3:]:
            assert len(line) == 81   # 79 plus two separators

    def test_max_rows(self):
        """
        Should be able to cap the number of rows
        """
        self.render_config_template(
            mysql_ports=[3306],
            mysql_max_row_length=79,
            mysql_max_rows=5,
            mysql_send_response=True
        )
        self.run_packetbeat(pcap="mysql_long_result.pcap",
                            debug_selectors=["mysqldetailed"])

        objs = self.read_output()
        assert len(objs) == 1
        res = objs[0]
        assert res["mysql.num_rows"] == 15

        lines = res["response"].strip().split("\n")
        assert len(lines) == 6    # 5 plus header

        for line in lines[3:]:
            assert len(line) == 81   # 79 plus two separators

    def test_larger_max_rows(self):
        """
        Should be able to cap the number of rows
        """
        self.render_config_template(
            mysql_ports=[3306],
            mysql_max_rows=2000,
            mysql_send_response=True
        )
        self.run_packetbeat(pcap="mysql_long_result.pcap",
                            debug_selectors=["mysqldetailed"])

        objs = self.read_output()
        assert len(objs) == 1
        res = objs[0]
        assert res["mysql.num_rows"] == 15

        lines = res["response"].strip().split("\n")
        assert len(lines) == 16    # 15 plus header

    def test_larger_than_100k(self):
        """
        Should work for MySQL messages larger than 100k bytes.
        """
        self.render_config_template(
            mysql_ports=[3306],
            mysql_send_response=True
        )
        self.run_packetbeat(pcap="mysql_long.pcap",
                            debug_selectors=["mysqldetailed"])

        objs = self.read_output()
        assert len(objs) == 1
        res = objs[0]
        assert res["mysql.num_rows"] == 400
