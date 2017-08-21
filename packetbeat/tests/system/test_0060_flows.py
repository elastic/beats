from packetbeat import (BaseTest, FLOWS_REQUIRED_FIELDS)
from pprint import PrettyPrinter
from datetime import datetime
import six


def pprint(x): return PrettyPrinter().pprint(x)


def check_fields(flow, fields):
    for k, v in six.iteritems(fields):
        assert flow[k] == v


def parse_timestamp(ts):
    return datetime.strptime(ts, "%Y-%m-%dT%H:%M:%S.%fZ")


class Test(BaseTest):

    def test_mysql_flow(self):
        self.render_config_template(
            flows=True,
            shutdown_timeout="1s",
        )
        self.run_packetbeat(
            pcap="mysql_long.pcap",
            debug_selectors=["*"])

        objs = self.read_output(
            types=["flow"],
            required_fields=FLOWS_REQUIRED_FIELDS)

        pprint(objs)
        assert len(objs) == 1
        check_fields(objs[0], {
            'final': True,
            'source.mac': '0a:00:27:00:00:00',
            'dest.mac': '08:00:27:76:d7:41',
            'dest.ip': '192.168.33.14',
            'source.ip': '192.168.33.1',
            'transport': 'tcp',
            'source.port': 60137,
            'dest.port': 3306,
            'source.stats.net_packets_total': 22,
            'source.stats.net_bytes_total': 1480,
            'dest.stats.net_packets_total': 10,
            'dest.stats.net_bytes_total': 181133,
        })

        start_ts = parse_timestamp(objs[0]['start_time'])
        last_ts = parse_timestamp(objs[0]['last_time'])
        assert last_ts > start_ts

    def test_memcache_udp_flow(self):
        self.render_config_template(
            flows=True,
            shutdown_timeout="1s",
        )
        self.run_packetbeat(
            pcap="memcache/memcache_bin_udp_counter_ops.pcap",
            debug_selectors=["*"])

        objs = self.read_output(
            types=["flow"],
            required_fields=FLOWS_REQUIRED_FIELDS)

        pprint(objs)
        assert len(objs) == 1
        check_fields(objs[0], {
            'final': True,
            'source.mac': 'ac:bc:32:77:41:0b',
            'dest.mac': '08:00:27:dd:3b:28',
            'source.ip': '192.168.188.37',
            'dest.ip': '192.168.188.38',
            'transport': 'udp',
            'source.port': 63888,
            'dest.port': 11211,
            'source.stats.net_packets_total': 3,
            'source.stats.net_bytes_total': 280,
        })

    def test_icmp4_ping(self):
        self.render_config_template(
            flows=True,
            shutdown_timeout="1s",
        )
        self.run_packetbeat(
            pcap="icmp/icmp4_ping_over_vlan.pcap",
            debug_selectors=["*"])

        objs = self.read_output(
            types=["flow"],
            required_fields=FLOWS_REQUIRED_FIELDS)

        pprint(objs)
        assert len(objs) == 1
        check_fields(objs[0], {
            'final': True,
            'source.mac': '00:00:00:00:00:01',
            'dest.mac': '00:00:00:00:00:02',
            'vlan': 10,
            'source.ip': '10.0.0.1',
            'dest.ip': '10.0.0.2',
            'icmp_id': 5,
            'source.stats.net_bytes_total': 50,
            'source.stats.net_packets_total': 1,
            'dest.stats.net_bytes_total': 50,
            'dest.stats.net_packets_total': 1,
        })

    def test_icmp6_ping(self):
        self.render_config_template(
            flows=True,
            shutdown_timeout="1s",
        )
        self.run_packetbeat(
            pcap="icmp/icmp6_ping_over_vlan.pcap",
            debug_selectors=["*"])

        objs = self.read_output(
            types=["flow"],
            required_fields=FLOWS_REQUIRED_FIELDS)

        pprint(objs)
        assert len(objs) == 1
        check_fields(objs[0], {
            'final': True,
            'source.mac': '00:00:00:00:00:01',
            'dest.mac': '00:00:00:00:00:02',
            'vlan': 10,
            'source.ipv6': '::1',
            'dest.ipv6': '::2',
            'icmp_id': 5,
            'source.stats.net_bytes_total': 70,
            'source.stats.net_packets_total': 1,
            'dest.stats.net_bytes_total': 70,
            'dest.stats.net_packets_total': 1,
        })
