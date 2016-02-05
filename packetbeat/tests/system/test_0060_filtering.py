from packetbeat import BaseTest


class Test(BaseTest):

    def test_http_filter(self):

        self.render_config_template(
	        http_send_all_headers=True,
            drop_fields=["http.request_headers"], 
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



    def test_http_fail_filter(self):

        self.render_config_template(
            http_send_all_headers=True,
            drop_fields=["http.response_headers.transfer-encoding"],
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

        print objs[0]

        assert "http.request_headers" in objs[0]
        assert "http.response_headers" in objs[0]

        # generic filtering fails to delete the htp.response_headers.transfer-encoding
        assert "transfer-encoding" in objs[0]["http.response_headers"]
