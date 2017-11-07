from packetbeat import BaseTest


class Test(BaseTest):

    def test_drop_map_fields(self):

        self.render_config_template(
            http_send_all_headers=True,
            processors=[{
                "drop_fields": {
                    "fields": ["http.request.headers"]
                },
            }]
        )

        self.run_packetbeat(pcap="http_minitwit.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output()

        assert len(objs) == 3
        assert all([o["type"] == "http" for o in objs])

        assert objs[0]["status"] == "OK"
        assert objs[1]["status"] == "OK"
        assert objs[2]["status"] == "Error"

        assert "http.request.headers" not in objs[0]
        assert "http.response.headers" in objs[0]

    def test_drop_fields_with_cond(self):

        self.render_config_template(
            http_send_all_headers=True,
            processors=[{
                "drop_fields": {
                    "fields": ["http.request.headers", "http.response.headers"],
                    "when": "equals.http.response.code: 200",
                },
            }]
        )

        self.run_packetbeat(pcap="http_minitwit.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output(required_fields=["@timestamp", "type"])

        assert len(objs) == 3
        assert all([o["type"] == "http" for o in objs])

        assert "http.request.headers" not in objs[0]
        assert "http.response.headers" not in objs[0]

        assert "status" in objs[0]
        assert "http.response.code" in objs[0]

        assert "http.request.headers" in objs[1]
        assert "http.response.headers" in objs[1]

        assert "http.request.headers" in objs[2]
        assert "http.response.headers" in objs[2]

    def test_include_fields_with_cond(self):

        self.render_config_template(
            http_send_request=True,
            http_send_response=True,
            processors=[{
                "include_fields": {
                    "fields": ["http"],
                    "when": "equals.http.code: 200",
                },
            }]
        )

        self.run_packetbeat(pcap="http_minitwit.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output(required_fields=["@timestamp", "type"])

        assert len(objs) == 3
        assert all([o["type"] == "http" for o in objs])

        assert "http.request.headers" in objs[0]
        assert "http.response.headers" in objs[0]

        assert "response" in objs[0]
        assert "request" in objs[0]

        assert "http.response.code" in objs[0]

        assert "request" in objs[1]
        assert "response" in objs[1]

        assert "request" in objs[2]
        assert "response" in objs[2]

    def test_drop_fields_with_cond_range(self):

        self.render_config_template(
            http_send_request=True,
            http_send_response=True,
            processors=[{
                "drop_fields": {
                    "fields": ["request", "response"],
                    "when": "range.http.response.code.lt: 300",
                },
            }]
        )

        self.run_packetbeat(pcap="http_minitwit.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output(required_fields=["@timestamp", "type"])

        assert len(objs) == 3
        assert all([o["type"] == "http" for o in objs])

        print(objs[0])
        assert "response" not in objs[0]
        assert "request" not in objs[0]

        assert "status" in objs[0]
        assert "http.response.code" in objs[0]

        assert "request" in objs[1]
        assert "response" in objs[1]

        assert "request" in objs[2]
        assert "response" in objs[2]

    def test_drop_event_with_cond(self):

        self.render_config_template(
            processors=[{
                "drop_event": {
                    "when": "range.http.response.code.lt: 300",
                },
            }]
        )

        self.run_packetbeat(pcap="http_minitwit.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output(required_fields=["@timestamp", "type"])

        print(objs)
        assert len(objs) == 2
        assert all([o["type"] == "http" for o in objs])

        assert all([o["http.response.code"] > 300 for o in objs])

    def test_drop_end_fields(self):

        self.render_config_template(
            http_send_all_headers=True,
            processors=[{
                "drop_fields": {
                    "fields": ["http.response.headers.transfer-encoding"]
                },
            }]
        )

        self.run_packetbeat(pcap="http_minitwit.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output()

        assert len(objs) == 3
        assert all([o["type"] == "http" for o in objs])

        assert objs[0]["status"] == "OK"
        assert objs[1]["status"] == "OK"
        assert objs[2]["status"] == "Error"

        assert "http.request.headers" in objs[0]
        assert "http.response.headers" in objs[0]

        # check if filtering deleted the
        # htp.response_headers.transfer-encoding
        assert "transfer-encoding" not in objs[0]["http.response.headers"]

    def test_drop_unknown_field(self):

        self.render_config_template(
            http_send_all_headers=True,
            processors=[{
                "drop_fields": {
                    "fields": ["http.response.headers.transfer-encoding-test"]
                },
            }]
        )

        self.run_packetbeat(pcap="http_minitwit.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output()

        assert len(objs) == 3
        assert all([o["type"] == "http" for o in objs])

        assert objs[0]["status"] == "OK"
        assert objs[1]["status"] == "OK"
        assert objs[2]["status"] == "Error"

        assert "http.request.headers" in objs[0]
        assert "http.response.headers" in objs[0]

        # check that htp.response_headers.transfer-encoding
        # still exists
        assert "transfer-encoding" in objs[0]["http.response.headers"]

    def test_drop_event(self):

        self.render_config_template(
            http_send_all_headers=True,
            processors=[{
                "drop_event": {
                    "when": "equals.status: OK",
                },
            }]
        )

        self.run_packetbeat(pcap="http_minitwit.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output()

        assert len(objs) == 1

    def test_include_empty_list(self):

        self.render_config_template(
            http_send_all_headers=True,
            # export all mandatory fields
            processors=[{
                "include_fields": {
                    "fields": [],
                },
            }]
        )

        self.run_packetbeat(pcap="http_minitwit.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output(
            required_fields=["@timestamp", "type"],
        )

        assert len(objs) == 3
        assert "http.request.headers" not in objs[0]
        assert "http.response.headers" not in objs[0]

    def test_drop_no_fields(self):
        self.render_config_template(
            http_send_all_headers=True,
            processors=[{
                "drop_fields": {
                    "fields": [],
                },
                # export all fields
            }]
        )

        self.run_packetbeat(pcap="http_minitwit.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output()

        assert len(objs) == 3
        assert all([o["type"] == "http" for o in objs])

        assert objs[0]["status"] == "OK"
        assert objs[1]["status"] == "OK"
        assert objs[2]["status"] == "Error"

    def test_drop_and_include_fields_failed_cond(self):
        self.render_config_template(
            http_send_all_headers=True,
            processors=[{
                "include_fields": {
                    "fields": ["http"],
                },
            }, {
                "drop_fields": {
                    "fields": ["http.request.headers", "http.response.headers"],
                    "when": "equals.status: OK",
                },
            }]
        )

        self.run_packetbeat(pcap="http_minitwit.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output(
            required_fields=["@timestamp", "type"],
        )

        assert len(objs) == 3
        assert all([o["type"] == "http" for o in objs])

        assert "http.request.headers" in objs[0]
        assert "http.response.headers" in objs[0]

        assert "http.request.headers" in objs[1]
        assert "http.response.headers" in objs[1]

    def test_drop_and_include_fields(self):

        self.render_config_template(
            http_send_all_headers=True,
            processors=[{
                "include_fields": {
                    "fields": ["http"],
                },
            }, {
                "drop_fields": {
                    "fields": ["http.request.headers", "http.response.headers"],
                    "when": "equals.http.response.code: 200",
                },
            }]
        )

        self.run_packetbeat(pcap="http_minitwit.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output(
            required_fields=["@timestamp", "type"],
        )

        assert len(objs) == 3
        assert all([o["type"] == "http" for o in objs])

        assert "http.request.headers" not in objs[0]
        assert "http.response.headers" not in objs[0]

        assert "http.request.headers" in objs[1]
        assert "http.response.headers" in objs[1]

    def test_condition_and(self):

        self.render_config_template(
            http_send_all_headers=True,
            processors=[{
                "include_fields": {
                    "fields": ["http"],
                    "when": """
                    and:
                      - equals.type: http
                      - equals.http.response.code: 200
                    """
                },
            }]
        )

        self.run_packetbeat(pcap="http_minitwit.pcap",
                            debug_selectors=["http", "httpdetailed", "processors"])
        objs = self.read_output(
            required_fields=["@timestamp", "type"],
        )

        assert len(objs) == 3
        assert all([o["type"] == "http" for o in objs])

        assert "method" not in objs[0]

    def test_condition_or(self):

        self.render_config_template(
            http_send_all_headers=True,
            processors=[{
                "drop_event": {
                    "when": """
                      or:
                        - equals.http.response.code: 404
                        - equals.http.response.code: 200
                    """
                },
            }]
        )

        self.run_packetbeat(pcap="http_minitwit.pcap",
                            debug_selectors=["http", "httpdetailed", "processors"])
        objs = self.read_output(
            required_fields=["@timestamp", "type"],
        )

        print(objs)
        assert len(objs) == 1
        assert all([o["type"] == "http" for o in objs])

    def test_condition_not(self):

        self.render_config_template(
            http_send_all_headers=True,
            processors=[{
                "drop_event": {
                    "when.not": "equals.http.response.code: 200",
                },
            }]
        )

        self.run_packetbeat(pcap="http_minitwit.pcap",
                            debug_selectors=["http", "httpdetailed", "processors"])
        objs = self.read_output(
            required_fields=["@timestamp", "type"],
        )

        assert len(objs) == 1
        assert all([o["type"] == "http" for o in objs])
