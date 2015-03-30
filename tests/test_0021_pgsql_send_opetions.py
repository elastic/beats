from pbtests.packetbeat import TestCase

"""
Tests for the pgsql's send_request and send_response
options.
"""


class Test(TestCase):

    def test_default_settings(self):
        """
        Should not include request_raw and response_raw by
        default.
        """
        self.render_config_template()
        self.run_packetbeat(pcap="pgsql_long_result.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        res = objs[0]

        assert "request_raw" not in res
        assert "response_raw" not in res

    def run_with_options(self, send_request, send_response):
        self.render_config_template(
            pgsql_send_request=send_request,
            pgsql_send_response=send_response,
        )
        self.run_packetbeat(pcap="pgsql_long_result.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        return objs[0]

    def test_send_request_only(self):
        res = self.run_with_options(True, False)
        assert "request_raw" in res
        assert len(res["request_raw"]) > 0
        assert "response_raw" not in res

    def test_send_response_only(self):
        res = self.run_with_options(False, True)
        assert "request_raw" not in res
        assert "response_raw" in res
        assert len(res["response_raw"]) > 0

    def test_both_off(self):
        res = self.run_with_options(False, False)
        assert "request_raw" not in res
        assert "response_raw" not in res

    def test_both_on(self):
        res = self.run_with_options(True, True)
        assert "request_raw" in res
        assert "response_raw" in res
