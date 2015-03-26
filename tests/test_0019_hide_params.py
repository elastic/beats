from pbtests.packetbeat import TestCase

"""
Tests for checking the hide_keywords options.
"""


class Test(TestCase):

    def test_http_hide_post(self):
        """
        Should be able to strip the password from
        a POST request.
        """
        self.render_config_template(
            http_send_all_headers=1,
            http_hide_keywords=["pass", "password"]
        )
        self.run_packetbeat(pcap="hide_secret_POST.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output()

        assert len(objs) == 1
        o = objs[0]
        assert o["type"] == "http"
        assert o["params"] == "pass=xxxxx&user=monica"

    def test_http_hide_get(self):
        """
        Should be able to strip the password from
        a GET request.
        """
        self.render_config_template(
            http_send_all_headers=1,
            http_hide_keywords=["pass", "password"]
        )
        self.run_packetbeat(pcap="hide_secret_GET.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output()

        assert len(objs) == 1
        o = objs[0]
        assert o["type"] == "http"
        assert o["params"] == "pass=xxxxx&user=monica"

    def test_http_hide_post_default(self):
        """
        Should be able to strip the password from
        a POST request.
        """
        self.render_config_template(
            http_send_all_headers=1
        )
        self.run_packetbeat(pcap="hide_secret_POST.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output()

        assert len(objs) == 1
        o = objs[0]
        assert o["type"] == "http"
        assert o["params"] == "pass=secret&user=monica"
