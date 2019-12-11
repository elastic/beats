from packetbeat import BaseTest

"""
Tests for the DNS protocol.
"""


class Test(BaseTest):

    def test_A(self):
        """
        Should correctly interpret an A query to google.com
        """
        self.render_config_template(
            dns_ports=[53],
        )
        self.run_packetbeat(pcap="dns_google_com.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert o["type"] == "dns"
        assert o["network.type"] == "ipv4"
        assert o["network.protocol"] == "dns"
        assert o["network.transport"] == "udp"
        assert o["network.bytes"] == 312
        assert "network.community_id" in o
        assert o["event.start"] == "2015-08-27T08:00:55.638Z"
        assert o["event.end"] == "2015-08-27T08:00:55.700Z"
        assert o["event.duration"] == 61782000
        assert o["client.ip"] == "192.168.238.68"
        assert o["source.ip"] == "192.168.238.68"
        assert o["client.port"] == 60893
        assert o["source.port"] == 60893
        assert o["source.bytes"] == 28
        assert o["server.ip"] == "192.168.238.1"
        assert o["destination.ip"] == "192.168.238.1"
        assert o["server.port"] == 53
        assert o["destination.port"] == 53
        assert o["destination.bytes"] == 284
        assert o["method"] == "QUERY"
        assert o["query"] == "class IN, type A, google.com"
        assert o["dns.question.type"] == "A"
        assert o["status"] == "OK"
        assert len(o["dns.answers"]) == 16
        assert all(x["type"] == "A" for x in o["dns.answers"])

    def test_A_not_found(self):
        """
        Should correctly interpret an A query to google.com
        """
        self.render_config_template(
            dns_ports=[53],
        )
        self.run_packetbeat(pcap="dns_not_found.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert o["type"] == "dns"
        assert o["network.protocol"] == "dns"
        assert o["network.transport"] == "udp"
        assert o["method"] == "QUERY"
        assert o["query"] == "class IN, type A, nothing.elastic.co"
        assert o["dns.question.type"] == "A"
        assert o["status"] == "Error"
        assert o["dns.response_code"] == "NXDOMAIN"
        assert o["dns.answers_count"] == 0
        assert o["dns.authorities_count"] == 1
        assert "dns.authorities" not in o   # include authorities defaults to 0

    def test_MX(self):
        """
        Should correctly interpret an MX query to elastic.co
        """
        self.render_config_template(
            dns_ports=[53],
        )
        self.run_packetbeat(pcap="dns_mx.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert o["type"] == "dns"
        assert o["network.protocol"] == "dns"
        assert o["network.transport"] == "udp"
        assert o["method"] == "QUERY"
        assert o["query"] == "class IN, type MX, elastic.co"
        assert o["dns.question.type"] == "MX"
        assert o["status"] == "OK"

    def test_NS(self):
        """
        Should correctly interpret an NS query to elastic.co
        """
        self.render_config_template(
            dns_ports=[53],
        )
        self.run_packetbeat(pcap="dns_ns.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert o["type"] == "dns"
        assert o["network.protocol"] == "dns"
        assert o["network.transport"] == "udp"
        assert o["method"] == "QUERY"
        assert o["query"] == "class IN, type NS, elastic.co"
        assert o["dns.question.type"] == "NS"
        assert o["status"] == "OK"

    def test_TXT(self):
        """
        Should correctly interpret an TXT query to elastic.co
        """
        self.render_config_template(
            dns_ports=[53],
        )
        self.run_packetbeat(pcap="dns_txt.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert o["type"] == "dns"
        assert o["network.protocol"] == "dns"
        assert o["network.transport"] == "udp"
        assert o["method"] == "QUERY"
        assert o["destination.ip"] == "8.8.8.8"
        assert o["server.ip"] == "8.8.8.8"
        assert o["query"] == "class IN, type TXT, elastic.co"
        assert o["dns.question.type"] == "TXT"
        assert o["status"] == "OK"
        assert len(o["dns.answers"]) == 2
        assert all(x["type"] == "TXT" for x in o["dns.answers"])
        assert "request" not in o
        assert "response" not in o

    def test_include_authorities(self):
        """
        Should include DNS authorities when configured.
        """
        self.render_config_template(
            dns_ports=[53],
            dns_include_authorities=True
        )

        self.run_packetbeat(pcap="dns_not_found.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert o["type"] == "dns"
        assert o["network.protocol"] == "dns"
        assert o["dns.authorities_count"] == 1
        assert "dns.authorities" in o
        assert len(o["dns.authorities"]) == 1

    def test_include_additionals(self):
        """
        Should include DNS authorities when configured.
        """
        self.render_config_template(
            dns_ports=[53],
            dns_include_additionals=True
        )

        self.run_packetbeat(pcap="dns_additional.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert o["type"] == "dns"
        assert o["network.protocol"] == "dns"
        assert o["dns.additionals_count"] == 1
        assert "dns.additionals" in o
        assert len(o["dns.additionals"]) == 1

    def test_send_request_response(self):
        """
        Should correctly interpret an TXT query to elastic.co
        """
        self.render_config_template(
            dns_ports=[53],
            dns_send_request=True,
            dns_send_response=True
        )
        self.run_packetbeat(pcap="dns_txt.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert o["type"] == "dns"
        assert o["network.protocol"] == "dns"
        assert o["network.transport"] == "udp"
        assert o["method"] == "QUERY"
        assert "request" in o
        assert "response" in o
        assert "elastic.co" in o["request"]
        assert "include:_spf.google.com" in o["response"]

    def test_tcp_axfr(self):
        """
        Should correctly interpret a TCP AXFR query
        """
        self.render_config_template(
            dns_ports=[53],
            dns_send_request=True,
            dns_send_response=True
        )
        self.run_packetbeat(pcap="dns_tcp_axfr.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert o["type"] == "dns"
        assert o["network.protocol"] == "dns"
        assert o["network.transport"] == "tcp"
        assert o["method"] == "QUERY"
        assert o["query"] == "class IN, type AXFR, etas.com"
        assert o["dns.question.type"] == "AXFR"
        assert o["status"] == "OK"
        assert len(o["dns.answers"]) == 4
        assert all("etas.com" in x["name"] for x in o["dns.answers"])

    def test_edns_dnssec(self):
        """
        Should correctly interpret a UDP edns with a DNSSEC RR
        """
        self.render_config_template(
            dns_ports=[53],
        )
        self.run_packetbeat(pcap="dns_udp_edns_ds.pcap")

        objs = self.read_output()
        assert len(objs) == 1
        o = objs[0]

        assert o["type"] == "dns"
        assert o["network.protocol"] == "dns"
        assert o["network.transport"] == "udp"
        assert o["method"] == "QUERY"
        assert o["query"] == "class IN, type DS, ietf.org"
        assert o["dns.question.type"] == "DS"
        assert o["status"] == "OK"
        assert o["dns.opt.do"] == True
        assert o["dns.opt.version"] == "0"
        assert o["dns.opt.udp_size"] == 4000
        assert o["dns.opt.ext_rcode"] == "NOERROR"
        assert len(o["dns.answers"]) == 3
        assert all("ietf.org" in x["name"] for x in o["dns.answers"])
