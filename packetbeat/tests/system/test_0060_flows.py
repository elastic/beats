from packetbeat import (BaseTest, FLOWS_REQUIRED_FIELDS)

from pprint import PrettyPrinter


pprint = lambda x: PrettyPrinter().pprint(x)


def check_fields(flow, fields):
    for k, v in fields.iteritems():
        assert flow[k] == v


class Test(BaseTest):
    def test_mysql_flow(self):
        self.render_config_template(
            flows=True,
        )
        self.run_packetbeat(
            pcap="mysql_long.pcap",
            wait_stop=1,
            debug_selectors=["*"])

        objs = self.read_output(
            types=["flow"],
            required_fields=FLOWS_REQUIRED_FIELDS)

        pprint(objs)
        assert len(objs) == 1
        check_fields(objs[0], {
            'final': True,
            'mac_source': '0a:00:27:00:00:00',
            'mac_dest': '08:00:27:76:d7:41',
            'ip4_dest': '192.168.33.14',
            'ip4_source': '192.168.33.1',
            'transport': 'tcp',
            'port_source': 60137,
            'port_dest': 3306,
            'stats_source.net_packets_total': 22,
            'stats_source.net_bytes_total': 1480,
            'stats_dest.net_packets_total': 10,
            'stats_dest.net_bytes_total': 181133,
        })

    def test_memcache_udp_flow(self):
        self.render_config_template(
            flows=True,
        )
        self.run_packetbeat(
            pcap="memcache/memcache_bin_udp_counter_ops.pcap",
            wait_stop=1,
            debug_selectors=["*"])

        objs = self.read_output(
            types=["flow"],
            required_fields=FLOWS_REQUIRED_FIELDS)

        pprint(objs)
        assert len(objs) == 1
        check_fields(objs[0], {
            'final': True,
            'mac_source': 'ac:bc:32:77:41:0b',
            'mac_dest': '08:00:27:dd:3b:28',
            'ip4_source': '192.168.188.37',
            'ip4_dest': '192.168.188.38',
            'transport': 'udp',
            'port_source': 63888,
            'port_dest': 11211,
            'stats_source.net_packets_total': 3,
            'stats_source.net_bytes_total': 280,
        })

    def test_icmp4_ping(self):
        self.render_config_template(
            flows=True,
        )
        self.run_packetbeat(
            pcap="icmp/icmp4_ping_over_vlan.pcap",
            wait_stop=1,
            debug_selectors=["*"])

        objs = self.read_output(
            types=["flow"],
            required_fields=FLOWS_REQUIRED_FIELDS)

        pprint(objs)
        assert len(objs) == 1
        check_fields(objs[0], {
            'final': True,
            'mac_source': '00:00:00:00:00:01',
            'mac_dest': '00:00:00:00:00:02',
            'vlan': 10,
            'ip4_source': '10.0.0.1',
            'ip4_dest': '10.0.0.2',
            'icmp_id': 5,
            'stats_source.net_bytes_total': 50,
            'stats_source.net_packets_total': 1,
            'stats_dest.net_bytes_total': 50,
            'stats_dest.net_packets_total': 1,
        })

    def test_icmp6_ping(self):
        self.render_config_template(
            flows=True,
        )
        self.run_packetbeat(
            pcap="icmp/icmp6_ping_over_vlan.pcap",
            wait_stop=1,
            debug_selectors=["*"])

        objs = self.read_output(
            types=["flow"],
            required_fields=FLOWS_REQUIRED_FIELDS)

        pprint(objs)
        assert len(objs) == 1
        check_fields(objs[0], {
            'final': True,
            'mac_source': '00:00:00:00:00:01',
            'mac_dest': '00:00:00:00:00:02',
            'vlan': 10,
            'ip6_source': '::1',
            'ip6_dest': '::2',
            'icmp_id': 5,
            'stats_source.net_bytes_total': 70,
            'stats_source.net_packets_total': 1,
            'stats_dest.net_bytes_total': 70,
            'stats_dest.net_packets_total': 1,
        })
