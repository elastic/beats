from packetbeat import BaseTest

import six

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
        assert o["http.response.code"] == 200
        assert o["http.response.phrase"] == "OK"
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
        assert objs[13]["http.response.code"] == 500

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

        assert "http.request.headers" in o
        assert "http.response.headers" in o

        self.render_config_template(
            http_ports=[8888],
            http_send_all_headers=True,
        )
        self.run_packetbeat(pcap="wsgi_loopback.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert "http.request.headers" in o
        assert "http.response.headers" in o
        assert o["http.request.headers"]["cache-control"] == "max-age=0"
        assert len(o["http.request.headers"]) > 0
        assert len(o["http.response.headers"]) > 0
        assert isinstance(o["http.response.headers"]["set-cookie"],
                          six.string_types)

        self.render_config_template(
            http_ports=[8888],
            http_send_headers=["User-Agent", "content-Type",
                               "x-forwarded-for"],
        )
        self.run_packetbeat(pcap="wsgi_loopback.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert "http.request.headers" in o
        assert "http.response.headers" in o
        assert len(o["http.request.headers"]) > 0
        assert len(o["http.response.headers"]) > 0
        assert "user-agent" in o["http.request.headers"]

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
        print(o)

        assert len(o["http.request.headers"]) > 0
        assert len(o["http.response.headers"]) > 0

        assert isinstance(o["http.request.headers"]["cookie"], dict)
        assert len(o["http.request.headers"]["cookie"]) == 6

        assert isinstance(o["http.response.headers"]["set-cookie"], dict)
        assert len(o["http.response.headers"]["set-cookie"]) == 4
