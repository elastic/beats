from packetbeat import BaseTest

"""
Tests for checking that HTTP query is filled.
"""


class Test(BaseTest):

    def test_http_post(self):
        """
        Should be able to parse the parameters from the HTTP POST request.
        """
        self.render_config_template()
        self.run_packetbeat(pcap="http_post.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output()

        assert len(objs) == 1
        o = objs[0]
        assert o["type"] == "http"
        assert o["query"] == "POST /register"

    def test_http_get(self):
        """
        Should be able to parse the parameters from the HTTP POST request.
        """
        self.render_config_template()
        self.run_packetbeat(pcap="http_url_params.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output()

        assert len(objs) == 1
        o = objs[0]
        assert o["type"] == "http"
        assert o["query"] == "GET /dashboard/transactions"
        assert o["http.request.params"] == "input=packetbeat&src_ip=192.35.243.1"
