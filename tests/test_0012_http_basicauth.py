from pbtests.packetbeat import TestCase
import re


class Test(TestCase):

    def test_http_auth(self):
        self.render_config_template(
            http_send_all_headers=1,
            http_strip_authorization=1
        )
        self.run_packetbeat(pcap="http_basicauth.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output()

        assert len(objs) == 1
        assert all([o["type"] == "http" for o in objs])
        assert all([o["http.request_headers"]["authorization"] == "*"
                   for o in objs])
        assert all([re.search("Authorization:\*+", o["request_raw"])
                   is not None for o in objs])
