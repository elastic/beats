from packetbeat import BaseTest

"""
Tests for trimming long results in pgsql.
"""


class Test(BaseTest):

    def test_default_settings(self):
        """
        Should store the entire rows but only
        10 rows with default settings.
        """
        self.render_config_template(
            pgsql_ports=[5432],
            pgsql_send_response=True
        )
        self.run_packetbeat(pcap="pgsql_long_result.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        res = objs[0]
        assert res["pgsql.num_rows"] == 15

        lines = res["response"].strip().split("\n")
        assert len(lines) == 11    # 10 plus header

        for line in lines[4:]:
            print line, len(line)
            assert len(line) == 237

    def test_max_row_length(self):
        """
        Should be able to cap the row length.
        """
        self.render_config_template(
            pgsql_ports=[5432],
            pgsql_send_response=True,
            pgsql_max_row_length=79
        )
        self.run_packetbeat(pcap="pgsql_long_result.pcap",
                            debug_selectors=["pgsqldetailed"])

        objs = self.read_output()
        assert len(objs) == 1
        res = objs[0]
        assert res["pgsql.num_rows"] == 15

        lines = res["response"].strip().split("\n")
        assert len(lines) == 11    # 10 plus header

        for line in lines[4:]:
            print line, len(line)
            assert len(line) == 83   # 79 plus two separators and two quotes

    def test_max_rows(self):
        """
        Should be able to cap the number of rows
        """
        self.render_config_template(
            pgsql_ports=[5432],
            pgsql_send_response=True,
            pgsql_max_row_length=79,
            pgsql_max_rows=5
        )
        self.run_packetbeat(pcap="pgsql_long_result.pcap",
                            debug_selectors=["pgsqldetailed"])

        objs = self.read_output()
        assert len(objs) == 1
        res = objs[0]
        assert res["pgsql.num_rows"] == 15

        lines = res["response"].strip().split("\n")
        assert len(lines) == 6    # 5 plus header

    def test_larger_max_rows(self):
        """
        Should be able to cap the number of rows
        """
        self.render_config_template(
            pgsql_ports=[5432],
            pgsql_send_response=True,
            pgsql_max_rows=2000
        )
        self.run_packetbeat(pcap="pgsql_long_result.pcap",
                            debug_selectors=["pgsqldetailed"])

        objs = self.read_output()
        assert len(objs) == 1
        res = objs[0]
        assert res["pgsql.num_rows"] == 15

        lines = res["response"].strip().split("\n")
        assert len(lines) == 16    # 15 plus header
