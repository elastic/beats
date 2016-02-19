from packetbeat import BaseTest

"""
Tests for parsing WSGI traffic.
"""


class Test(BaseTest):

    def test_long_answer(self):
        self.render_config_template(
            http_ports=[8888]
        )
        self.run_packetbeat(pcap="wsgi_loopback.pcap")

        objs = self.read_output()
        assert len(objs) == 1

        o = objs[0]
        assert o["type"] == "http"
        assert o["client_port"] == 46249
        assert o["port"] == 8888
        assert o["status"] == "OK"
        assert o["method"] == "GET"
        assert o["path"] == "/"
        assert o["http.code"] == 200
        assert o["http.phrase"] == "OK"
        assert "request" not in objs[0]
        assert "response" not in objs[0]

    def test_drum_interraction(self):
        self.render_config_template(
            http_ports=[8888]
        )
        self.run_packetbeat(pcap="wsgi_drum.pcap",
                            debug_selectors=["tcp", "http", "protos"])

        objs = self.read_output()
        assert len(objs) == 16

        assert all([o["type"] == "http" for o in objs])
        assert all([o["port"] == 8888 for o in objs])

        assert all([o["status"] == "OK" for i, o in enumerate(objs)
                    if i != 13])

        assert objs[13]["status"] == "Error"
        assert objs[13]["path"] == "/comment/"
        assert objs[13]["http.code"] == 500

    def test_send_options(self):
        """
        Should put request and response in the output
        when requested.
        """
        self.render_config_template(
            http_ports=[8888],
            http_send_response=True,
            http_send_request=True,
        )
        self.run_packetbeat(pcap="wsgi_loopback.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        assert "request" in objs[0]
        assert "response" in objs[0]

    def test_include_body_for(self):
        self.render_config_template(
            http_ports=[8888],
            http_send_headers=["content-type"],
            http_include_body_for=["text/html"],
            http_send_response=True
        )
        self.run_packetbeat(pcap="wsgi_loopback.pcap")

        objs = self.read_output()
        assert len(objs) == 1

        assert len(objs[0]["response"]) > 20000

    def test_send_headers_options(self):
        self.render_config_template(
            http_ports=[8888],
        )
        self.run_packetbeat(pcap="wsgi_loopback.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert "http.requestHeaders" not in o
        assert "http.responseHeaders" not in o

        self.render_config_template(
            http_ports=[8888],
            http_send_all_headers=True,
        )
        self.run_packetbeat(pcap="wsgi_loopback.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert "http.requestHeaders" in o
        assert "http.responseHeaders" in o
        assert o["http.requestHeaders"]["cache-control"] == "max-age=0"
        assert len(o["http.requestHeaders"]) == 9
        assert len(o["http.responseHeaders"]) == 7
        assert isinstance(o["http.responseHeaders"]["set-cookie"],
                          basestring)

        self.render_config_template(
            http_ports=[8888],
            http_send_headers=["User-Agent", "content-Type",
                               "x-forwarded-for"],
        )
        self.run_packetbeat(pcap="wsgi_loopback.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert "http.requestHeaders" in o
        assert "http.responseHeaders" in o
        assert len(o["http.requestHeaders"]) == 1
        assert len(o["http.responseHeaders"]) == 1
        assert "user-agent" in o["http.requestHeaders"]

    def test_split_cookie(self):
        self.render_config_template(
            http_ports=[8888],
            http_send_all_headers=True,
            http_split_cookie=True,
        )
        self.run_packetbeat(pcap="wsgi_loopback.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert len(o["http.requestHeaders"]) == 9
        assert len(o["http.responseHeaders"]) == 7

        assert isinstance(o["http.requestHeaders"]["cookie"], dict)
        assert len(o["http.requestHeaders"]["cookie"]) == 6

        assert isinstance(o["http.responseHeaders"]["set-cookie"], dict)
        assert len(o["http.responseHeaders"]["set-cookie"]) == 4
