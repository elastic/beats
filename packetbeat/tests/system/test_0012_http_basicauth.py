from packetbeat import BaseTest
import re

"""
Tests for removing base64-encoded authentication information
"""


class Test(BaseTest):

    def test_http_auth_headers(self):
        self.render_config_template(
            dns_ports=[],       # disable dns because the pcap
                                # contains the DNS query
            http_send_all_headers=1,
            http_redact_authorization=1,
            http_ports=[80]
        )
        self.run_packetbeat(pcap="http_basicauth.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output()

        assert len(objs) >= 1
        assert all([o["type"] == "http" for o in objs])
        assert all([o["http.request.headers"]["authorization"] == "*"
                    is not None for o in objs])

    def test_http_auth_raw(self):
        self.render_config_template(
            dns_ports=[],       # disable dns because the pcap
                                # contains the DNS query
            http_redact_authorization=1,
            http_send_request=1,
            http_ports=[80]
        )
        self.run_packetbeat(pcap="http_basicauth.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output()

        assert len(objs) >= 1
        assert all([o["type"] == "http" for o in objs])
        assert all([re.search("[Aa]uthorization:\*+", o["request"])
                    is not None for o in objs])
