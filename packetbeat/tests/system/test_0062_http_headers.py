from packetbeat import BaseTest

"""
Tests for checking if the headers from HTTP request/response are exported correctly.
"""


class Test(BaseTest):

    def test_http_send_headers(self):
        """
        Check that content-length and content-type are sent even if they are not set under send_headers option.
        """
        self.render_config_template(
            http_send_headers=["host"],
        )
        self.run_packetbeat(pcap="http_post.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output()

        assert len(objs) == 1
        o = objs[0]
        print(o)
        assert o["type"] == "http"

        assert len(o["http.request.headers"]) > 0
        assert "content-length" in o["http.request.headers"]
        assert "content-type" in o["http.request.headers"]

        assert len(o["http.response.headers"]) > 0
        assert "content-length" in o["http.response.headers"]
        assert "content-type" in o["http.response.headers"]

    def test_http_send_all_headers(self):
        """
        Check that all headers are sent.
        """
        self.render_config_template(
            http_send_all_headers=True,
        )
        self.run_packetbeat(pcap="http_post.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output()

        assert len(objs) == 1
        o = objs[0]
        assert o["type"] == "http"

        assert len(o["http.request.headers"]) > 0
        assert "content-length" in o["http.request.headers"]
        assert "content-type" in o["http.request.headers"]
        assert "host" in o["http.request.headers"]
        assert "user-agent" in o["http.request.headers"]

        assert len(o["http.response.headers"]) > 0
        assert "content-length" in o["http.response.headers"]
        assert "content-type" in o["http.response.headers"]
        assert "date" in o["http.response.headers"]
        assert "connection" in o["http.response.headers"]
        assert "server" in o["http.response.headers"]
