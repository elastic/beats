from packetbeat import BaseTest


class Test(BaseTest):

    def test_http_sample(self):
        self.render_config_template()
        self.run_packetbeat(pcap="http_minitwit.pcap",
                            debug_selectors=["http", "httpdetailed"])
        objs = self.read_output()

        assert len(objs) == 3
        assert all([o["type"] == "http" for o in objs])
        assert all([o["client.ip"] == "192.168.1.104" for o in objs])
        assert all([o["client.port"] == 54742 for o in objs])
        assert all([o["server.ip"] == "192.168.1.110" for o in objs])
        assert all([o["server.port"] == 80 for o in objs])

        assert all(["network.bytes" in o for o in objs])
        assert all([o["network.type"] == "ipv4" for o in objs])
        assert all([o["network.transport"] == "tcp" for o in objs])
        assert all([o["network.protocol"] == "http" for o in objs])
        assert all(["network.community_id" in o for o in objs])

        assert all(["event.start" in o for o in objs])
        assert all(["event.end" in o for o in objs])
        assert all(["event.duration" in o for o in objs])

        assert all(["http.request.method" in o for o in objs])
        assert all(["http.request.bytes" in o for o in objs])
        assert all(["http.response.bytes" in o for o in objs])
        assert all(["http.response.status_code" in o for o in objs])
        assert all(["http.response.status_phrase" in o for o in objs])

        assert all(["url.full" in o for o in objs])

        assert all(["user_agent.original" in o for o in objs])

        assert objs[0]["status"] == "OK"
        assert objs[1]["status"] == "OK"
        assert objs[2]["status"] == "Error"

        assert all(["client.bytes" in o for o in objs])
        assert all(["server.bytes" in o for o in objs])

        assert objs[0]["client.bytes"] == 364
        assert objs[0]["server.bytes"] == 1000

        assert objs[1]["client.bytes"] == 471
        assert objs[1]["server.bytes"] == 234

        assert objs[2]["client.bytes"] == 289
        assert objs[2]["server.bytes"] == 396
