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
            'flow.final': True,
            'source.mac': '0a:00:27:00:00:00',
            'destination.mac': '08:00:27:76:d7:41',
            'destination.ip': '192.168.33.14',
            'source.ip': '192.168.33.1',
            'network.transport': 'tcp',
            'source.port': 60137,
            'destination.port': 3306,
            'source.packets': 22,
            'source.bytes': 1480,
            'destination.packets': 10,
            'destination.bytes': 181133,
        })

        start_ts = parse_timestamp(objs[0]['event.start'])
        last_ts = parse_timestamp(objs[0]['event.end'])
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
            'flow.final': True,
            'source.mac': 'ac:bc:32:77:41:0b',
            'destination.mac': '08:00:27:dd:3b:28',
            'source.ip': '192.168.188.37',
            'destination.ip': '192.168.188.38',
            'network.transport': 'udp',
            'source.port': 63888,
            'destination.port': 11211,
            'source.packets': 3,
            'source.bytes': 280,
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
            'flow.final': True,
            'source.mac': '00:00:00:00:00:01',
            'destination.mac': '00:00:00:00:00:02',
            'flow.vlan': 10,
            'source.ip': '10.0.0.1',
            'destination.ip': '10.0.0.2',
            'source.bytes': 50,
            'source.packets': 1,
            'destination.bytes': 50,
            'destination.packets': 1,
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
            'flow.final': True,
            'flow.vlan': 10,
            'source.mac': '00:00:00:00:00:01',
            'source.ip': '::1',
            'source.bytes': 70,
            'source.packets': 1,
            'destination.mac': '00:00:00:00:00:02',
            'destination.ip': '::2',
            'destination.bytes': 70,
            'destination.packets': 1,
            'network.bytes': 140,
            'network.packets': 2,
        })

    def test_q_in_q_flow(self):
        self.render_config_template(
            flows=True,
            shutdown_timeout="1s",
        )
        self.run_packetbeat(
            pcap="802.1q-q-in-q-icmp.pcap",
            debug_selectors=["*"])

        objs = self.read_output(
            types=["flow"],
            required_fields=FLOWS_REQUIRED_FIELDS)

        pprint(objs)
        assert len(objs) == 1
        check_fields(objs[0], {
            'flow.final': True,
            'flow.vlan': [101, 600],
            'source.ip': '192.168.1.1',
            'source.bytes': 82,
            'source.packets': 1,
            'source.mac': '08:00:27:3d:25:4e',
            'destination.mac': '1c:af:f7:70:ed:7c',
            'destination.ip': '192.168.1.2',
            'network.bytes': 82,
            'network.packets': 1,
        })
