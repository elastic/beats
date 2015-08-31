from pbtests.packetbeat import TestCase
import re


class Test(TestCase):

    def test_http_auth(self):
        self.render_config_template(
            dns_ports=[],       # disable dns because the pcap
                                # contains the DNS query
            http_send_all_headers=1,
            http_strip_authorization=1,
            http_send_request=True
        )
        self.run_packetbeat(pcap="http_basicauth.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output()

        assert len(objs) == 1
        assert all([o["type"] == "http" for o in objs])
        assert all([o["http.request_headers"]["authorization"] == "*"
                   for o in objs])
        assert all([re.search("Authorization:\*+", o["request"])
                   is not None for o in objs])
