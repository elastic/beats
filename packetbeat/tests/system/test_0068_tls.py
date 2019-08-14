from packetbeat import BaseTest


class Test(BaseTest):

    def test_tls12(self):
        self.render_config_template()
        self.run_packetbeat(pcap="http2-16-ssl.pcap",
                            debug_selectors=["tls"])
        objs = self.read_output()

        assert len(objs) == 1
        o = objs[0]

        assert o["type"] == "tls"
        assert o["source.ip"] == "127.0.0.1"
        assert o["source.port"] == 60883
        assert o["destination.ip"] == "127.0.0.1"
        assert o["destination.port"] == 443
        assert o["destination.domain"] == "localhost"

        assert "network.bytes" not in o
        assert o["network.type"] == "ipv4"
        assert o["network.transport"] == "tcp"
        assert o["network.protocol"] == "tls"
        assert "network.community_id" in o

        assert "event.start" in o
        assert "event.end" in o
        assert "event.duration" in o

        assert o["status"] == "OK"

        assert o["tls.fingerprints.ja3.hash"] == "ba7a226ea102737ecfa8959f26b28b95"
        assert o["tls.version"] == "TLS 1.2"
        assert o["tls.client_hello.extensions.server_name_indication"] == ["localhost"]
        assert o["tls.server_certificate.subject.common_name"] == "localhost"
