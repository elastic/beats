from packetbeat import BaseTest


def check_event(event, expected):
    for key in expected:
        assert key in event, "key '{0}' not found in event".format(key)
        assert event[key] == expected[key],\
            "key '{0}' has value '{1}', expected '{2}'".format(key,
                                                               event[key],
                                                               expected[key])


class Test(BaseTest):

    def test_unmatched_response(self):
        """
        Unmatched response in stream
        """

        self.render_config_template(
            http_ports=[8080],
        )
        self.run_packetbeat(pcap="http_unmatched.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output()

        assert len(objs) == 2

        check_event(objs[0], {
            "type": "http",
            "status": "Error",
            "http.response.code": 404,
            "notes": ["Unmatched response"]})

        check_event(objs[1], {
            "type": "http",
            "http.response.code": 200,
            "http.request.headers": {"content-length": 0},
            "status": "OK"})

    def test_unmatched_request(self):
        """
        Unmatched request due to timeout (15s)
        """

        self.render_config_template(
            http_ports=[8080],
            http_transaction_timeout="1s",
        )
        self.run_packetbeat(pcap="http_unmatched_timeout.pcap",
                            debug_selectors=["http", "httpdetailed"],
                            real_time=True)
        objs = self.read_output()
        print objs
        assert len(objs) == 1
        check_event(objs[0], {
            "type": "http",
            "status": "Error",
            "query": "GET /something",
            "notes": ["Unmatched request"]})
