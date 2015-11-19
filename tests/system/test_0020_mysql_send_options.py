from pbtests.packetbeat import TestCase

"""
Tests for the mysql's send_request and send_response
options.
"""


class Test(TestCase):

    def test_default_settings(self):
        """
        Should not include request and response by
        default.
        """
        self.render_config_template(
            mysql_ports=[3306],
        )
        self.run_packetbeat(pcap="mysql_long_result.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        res = objs[0]

        assert "request" not in res
        assert "response" not in res

    def run_with_options(self, send_request, send_response):
        self.render_config_template(
            mysql_ports=[3306],
            mysql_send_request=send_request,
            mysql_send_response=send_response,
        )
        self.run_packetbeat(pcap="mysql_long_result.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        return objs[0]

    def test_send_request_only(self):
        res = self.run_with_options(True, False)
        assert "request" in res
        assert len(res["request"]) > 0
        assert "response" not in res

    def test_send_response_only(self):
        res = self.run_with_options(False, True)
        assert "request" not in res
        assert "response" in res
        assert len(res["response"]) > 0

    def test_both_off(self):
        res = self.run_with_options(False, False)
        assert "request" not in res
        assert "response" not in res

    def test_both_on(self):
        res = self.run_with_options(True, True)
        assert "request" in res
        assert "response" in res
