from packetbeat import BaseTest

"""
Tests for checking if the body is exported correctly when send_body_for is set.
"""


class Test(BaseTest):

    def test_include_body(self):
        """
        Check that the http body is exported only for some http messages that have the
        content type in the list defined by include_body_for.
        """
        self.render_config_template(
            http_include_body_for=["x-www-form-urlencoded"],
            http_send_response=True,
        )
        self.run_packetbeat(pcap="http_post.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output()

        assert len(objs) == 1
        o = objs[0]

        assert o["type"] == "http"

        assert o["http.request.headers"]["content-type"] == "application/x-www-form-urlencoded"
        assert o["http.response.headers"]["content-type"] == "text/html; charset=utf-8"

        assert len(o["http.request.body"]) > 0
        assert "http.response.body" not in o

        # without body
        assert len(o["response"]) == 172

        assert "request" not in o

    def test_include_body_for_both_request_response(self):
        """
        Check that the http body is exported only for some http messages that have the
        content type in the list defined by include_body_for.
        """
        self.render_config_template(
            http_include_body_for=["x-www-form-urlencoded", "text/html"],
        )
        self.run_packetbeat(pcap="http_post.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output()

        assert len(objs) == 1
        o = objs[0]

        assert o["type"] == "http"

        assert o["http.request.headers"]["content-type"] == "application/x-www-form-urlencoded"
        assert o["http.response.headers"]["content-type"] == "text/html; charset=utf-8"

        assert len(o["http.request.body"]) > 0
        assert len(o["http.response.body"]) > 0

        assert "request" not in o
        assert "response" not in o

    def test_wrong_content_type(self):
        """
        Check if the body is exported for both request and response.
        Also checks that http.request.params is exported.
        """
        self.render_config_template(
            http_include_body_for=["x-www-form-urlencoded", "json"],
            http_ports=[80, 8080, 8000, 5000, 8002, 9200],
        )
        self.run_packetbeat(pcap="http_post_json.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()

        assert len(objs) == 1
        o = objs[0]
        print o

        assert o["type"] == "http"

        assert o["http.request.headers"]["content-type"] == "application/x-www-form-urlencoded; charset=UTF-8"
        assert o["http.response.headers"]["content-type"] == "application/json; charset=UTF-8"

        assert o["http.request.params"] == "%7B+%22query%22%3A+%7B+%22match_all%22%3A+%7B%7D%7D%7D%0A="
        assert len(o["http.request.body"]) > 0
        assert len(o["http.response.body"]) > 0

        assert "request" not in o
        assert "response" not in o

    def test_large_body(self):
        """
        Checks that the transaction is still created if the
        message is larger than the max_message_size.
        """
        self.render_config_template(
            http_include_body_for=["binary"],
            http_ports=[8000],
            http_max_message_size=1024
        )
        self.run_packetbeat(pcap="http_get_2k_file.pcap",
                            debug_selectors=["*"])
        objs = self.read_output()

        assert len(objs) == 1
        o = objs[0]
        print len(o["http.response.body"])

        # response body should be included but trimmed
        assert len(o["http.response.body"]) < 2000
        assert len(o["http.response.body"]) > 500
