from packetbeat import BaseTest


class Test(BaseTest):

    def test_drop_map_fields(self):

        self.render_config_template(
            http_send_all_headers=True,
            drop_fields=["http.request_headers"],
            # export all fields
            include_fields=None,
            filter_enabled=True,
        )

        self.run_packetbeat(pcap="http_minitwit.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output()

        assert len(objs) == 3
        assert all([o["type"] == "http" for o in objs])

        assert objs[0]["status"] == "OK"
        assert objs[1]["status"] == "OK"
        assert objs[2]["status"] == "Error"

        assert "http.request_headers" not in objs[0]
        assert "http.response_headers" in objs[0]

    def test_drop_end_fields(self):

        self.render_config_template(
            http_send_all_headers=True,
            drop_fields=["http.response_headers.transfer-encoding"],
            # export all fields
            include_fields=None,
            filter_enabled=True,
        )

        self.run_packetbeat(pcap="http_minitwit.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output()

        assert len(objs) == 3
        assert all([o["type"] == "http" for o in objs])

        assert objs[0]["status"] == "OK"
        assert objs[1]["status"] == "OK"
        assert objs[2]["status"] == "Error"

        assert "http.request_headers" in objs[0]
        assert "http.response_headers" in objs[0]

        # check if filtering deleted the
        # htp.response_headers.transfer-encoding
        assert "transfer-encoding" not in objs[0]["http.response_headers"]

    def test_drop_unknown_field(self):

        self.render_config_template(
            http_send_all_headers=True,
            drop_fields=["http.response_headers.transfer-encoding-test"],
            # export all fields
            include_fields=None,
            filter_enabled=True,
        )

        self.run_packetbeat(pcap="http_minitwit.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output()

        assert len(objs) == 3
        assert all([o["type"] == "http" for o in objs])

        assert objs[0]["status"] == "OK"
        assert objs[1]["status"] == "OK"
        assert objs[2]["status"] == "Error"

        assert "http.request_headers" in objs[0]
        assert "http.response_headers" in objs[0]

        # check that htp.response_headers.transfer-encoding
        # still exists
        assert "transfer-encoding" in objs[0]["http.response_headers"]

    def test_include_empty_list(self):

        self.render_config_template(
            http_send_all_headers=True,
            # export all mandatory fields
            include_fields=[],
            filter_enabled=True,
        )

        self.run_packetbeat(pcap="http_minitwit.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output(
            required_fields=["@timestamp", "type"],
        )

        assert len(objs) == 3
        assert "http.request_headers" not in objs[0]
        assert "http.response_headers" not in objs[0]

    def test_drop_no_fields(self):

        self.render_config_template(
            http_send_all_headers=True,
            drop_fields=[],
            # export all fields
            include_fields=None,
            filter_enabled=True,
        )

        self.run_packetbeat(pcap="http_minitwit.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output()

        assert len(objs) == 3
        assert all([o["type"] == "http" for o in objs])

        assert objs[0]["status"] == "OK"
        assert objs[1]["status"] == "OK"
        assert objs[2]["status"] == "Error"
