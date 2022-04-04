import json
import operator
import platform
import random
import socket
import struct
import time
import unittest
from auditbeat_xpack import *


def is_root():
    if 'geteuid' not in dir(os):
        return False
    euid = os.geteuid()
    return euid == 0


def is_version_below(version, target):
    t = list(map(int, target.split('.')))
    v = list(map(int, version.split('.')))
    v += [0] * (len(t) - len(v))
    for i in range(len(t)):
        if v[i] != t[i]:
            return v[i] < t[i]
    return False


# Require Linux greater or equal than 2.6.32 and 386/amd64 platform
def is_platform_supported():
    p = platform.platform().split('-')
    if p[0] != 'Linux':
        return False
    if is_version_below(p[1], '2.6.32'):
        return False
    return {'i386', 'i686', 'x86_64', 'amd64'}.intersection(p)


def enable_ipv6_loopback():
    f = open('/proc/sys/net/ipv6/conf/lo/disable_ipv6', 'wb')
    f.write(b'0\n')
    f.close()


@unittest.skipUnless(is_platform_supported(), "Requires Linux 2.6.32+ and 386/amd64 arch")
@unittest.skipUnless(is_root(), "Requires root")
class Test(AuditbeatXPackTest):
    def test_tcp_ipv4(self):
        """
        test TCP IPv4 flow
        """
        self.with_runner(TCP4TestCase())

    def test_udp_ipv4(self):
        """
        test UDP IPv4 flow
        """
        self.with_runner(UDP4TestCase())

    def test_connected_udp_ipv4(self):
        """
        test connected UDP IPv4 flow
        """
        self.with_runner(ConnectedUDP4TestCase())

    def test_connected_udp_ipv6(self):
        """
        test connected UDP IPv6 flow
        """
        self.with_runner(ConnectedUDP6TestCase())

    def test_udp_ipv6(self):
        """
        test UDP IPv6 flow
        """
        self.with_runner(UDP6TestCase())

    def test_multi_udp_upv4(self):
        """
        test multiple destination UDP IPv4 flows
        """
        self.with_runner(MultiUDP4TestCase())

    def test_udp_ipv6_disabled(self):
        """
        test IPv4/UDP with IPv6 disabled
        """
        self.with_runner(MultiUDP4TestCase(),
                         extra_conf={'socket.enable_ipv6': False})

    def test_tcp_ipv6_disabled(self):
        """
        test IPv4/TCP with IPv6 disabled
        """
        self.with_runner(TCP4TestCase(),
                         extra_conf={'socket.enable_ipv6': False})

    def test_dns_enrichment(self):
        """
        test DNS enrichment
        """
        self.with_runner(DNSTestCase())

    def test_no_dns_enrichment(self):
        """
        test DNS enrichment disabled
        """
        self.with_runner(
            DNSTestCase(enabled=False), extra_conf={'socket.dns.enabled': False})

    def test_dns_long_request(self):
        """
        test DNS enrichment of long request
        This test makes sure that DNS information is kept long after the
        DNS request has been performed, even if the internal DNS state
        is expired.
        """
        self.with_runner(
            DNSTestCase(delay_seconds=10),
            extra_conf={
                'socket.flow_inactive_timeout': '2s'
            })

    def test_dns_udp_ipv6(self):
        """
        test DNS enrichment of UDP/IPv6 session
        """
        self.with_runner(DNSTestCase(network="ipv6", transport="udp"))

    def test_dns_unidirectional_udp(self):
        """
        test DNS enrichment of unidirectional UDP
        """
        self.with_runner(DNSTestCase(transport="udp", bidirectional=False))

    def with_runner(self, test, extra_conf=dict()):
        enable_ipv6_loopback()
        conf = {
            "socket.flow_inactive_timeout": "2s",
            "socket.flow_termination_timeout": "5s",
            "socket.development_mode": "true",
        }
        conf.update(extra_conf)
        self.render_config_template(modules=[{
            "name": "system",
            "datasets": ["socket"],
            "extras": conf,
        }])
        proc = self.start_beat()
        try:
            try:
                self.wait_until(lambda: self.log_contains('system/socket dataset is running.'),
                                max_timeout=60)
            except Exception as e:
                raise Exception('Auditbeat failed to start start').with_traceback(sys.exc_info()[2])
            self.execute(test)
        finally:
            proc.check_kill_and_wait()

    def noop(self):
        pass

    def execute(self, test):
        cleanup = self.noop
        if hasattr(test, 'cleanup'):
            cleanup = test.cleanup

        if hasattr(test, 'setup'):
            test.setup()

        try:
            test.run()
        except Exception:
            cleanup()
            raise

        cleanup()

        try:
            self.wait_until(lambda: self.output_lines() > 0, max_timeout=15)
        except Exception as e:
            raise Exception('No output received form Auditbeat').with_traceback(sys.exc_info()[2])

        expected = test.expected()
        found = False
        try:
            self.wait_until(lambda: expected.match(self.flattened_output()), max_timeout=15)
            found = True
        finally:
            assert found, "The events in: {} don't match the condition: {}".format(
                pretty_print_json(list(self.flattened_output())),
                expected
            )

    def flattened_output(self):
        return [self.flatten_object(x, {}) for x in self.read_output_json()]


def pretty_print_json(d):
    return json.dumps(d, indent=3, default=lambda o: o.to_json(), sort_keys=True)


def is_length(a, n):
    try:
        return len(a) == n
    except TypeError:
        return False


class TCP4TestCase:
    def __init__(self):
        pass

    def run(self):
        client, self.client_addr = socket_ipv4(socket.SOCK_STREAM, socket.IPPROTO_TCP)
        server, self.server_addr = socket_ipv4(socket.SOCK_STREAM, socket.IPPROTO_TCP)
        server.listen(8)
        client.connect(self.server_addr)
        acc, _ = server.accept()
        acc.send(b'Hello there\n')
        msg = client.recv(64)
        client.send(bytes('"{}" what\n'.format(msg), "utf-8"))
        msg = acc.recv(64)
        acc.close()
        server.close()
        client.close()

    def expected(self):
        return HasEvent({
            "source.ip": self.client_addr[0],
            "source.port": self.client_addr[1],
            "source.bytes": Comparison(operator.gt, 20),
            "source.packets": Comparison(operator.gt, 4),
            "client.ip": self.client_addr[0],
            "client.port": self.client_addr[1],
            "destination.ip": self.server_addr[0],
            "destination.port": self.server_addr[1],
            "destination.bytes": Comparison(operator.gt, 20),
            "destination.packets": Comparison(operator.gt, 4),
            "server.ip": self.server_addr[0],
            "server.port": self.server_addr[1],
            "network.transport": "tcp",
            "network.type": "ipv4",
            "network.direction": "egress",
            "group.id": str(os.getgid()),
            "user.id": str(os.getuid()),
            "process.pid": os.getpid(),
            "process.entity_id": Comparison(is_length, 16),
        })


class UDP4TestCase:
    def __init__(self):
        pass

    def run(self):
        client, self.client_addr = socket_ipv4(socket.SOCK_DGRAM, socket.IPPROTO_UDP)
        server, self.server_addr = socket_ipv4(socket.SOCK_DGRAM, socket.IPPROTO_UDP)
        for i in range(3):
            client.sendto(bytes('Hello there {}'.format(i), "utf-8"), self.server_addr)
            msg, _ = server.recvfrom(64)
        server.sendto(b'howdy', self.client_addr)
        msg, _ = client.recvfrom(64)
        client.close()
        server.close()

    def expected(self):
        return HasEvent({
            "client.ip": self.client_addr[0],
            "client.port": self.client_addr[1],
            "destination.bytes": Comparison(operator.gt, 30),
            "destination.ip": self.server_addr[0],
            "destination.packets": 1,
            "destination.port": self.server_addr[1],
            "group.id": str(os.getgid()),
            "network.direction": "egress",
            "network.packets": 4,
            "network.transport": "udp",
            "network.type": "ipv4",
            "process.pid": os.getpid(),
            "process.entity_id": Comparison(is_length, 16),
            "server.ip": self.server_addr[0],
            "server.port": self.server_addr[1],
            "source.bytes": Comparison(operator.gt, 90),
            "source.ip": self.client_addr[0],
            "source.packets": 3,
            "source.port": self.client_addr[1],
            "user.id": str(os.getuid()),
        })


class ConnectedUDP4TestCase:
    def __init__(self):
        pass

    def run(self):
        client, self.client_addr = socket_ipv4(socket.SOCK_DGRAM, socket.IPPROTO_UDP)
        server, self.server_addr = socket_ipv4(socket.SOCK_DGRAM, socket.IPPROTO_UDP)
        client.connect(self.server_addr)
        server.connect(self.client_addr)
        for i in range(5):
            server.send(bytes('Hello there {}'.format(i), "utf-8"))
            msg = client.recv(64)
        client.send(b'howdy')
        msg = server.recv(64)
        client.send(b'bye')
        msg = server.recv(64)
        client.close()
        server.close()

    def expected(self):
        return HasEvent({
            "server.ip": self.client_addr[0],
            "server.port": self.client_addr[1],
            "source.bytes": Comparison(operator.gt, 150),
            "source.ip": self.server_addr[0],
            "source.packets": 5,
            "source.port": self.server_addr[1],
            "group.id": str(os.getgid()),
            "network.direction": "ingress",
            "network.packets": 7,
            "network.transport": "udp",
            "network.type": "ipv4",
            "process.pid": os.getpid(),
            "process.entity_id": Comparison(is_length, 16),
            "client.ip": self.server_addr[0],
            "client.port": self.server_addr[1],
            "destination.bytes": Comparison(operator.gt, 60),
            "destination.ip": self.client_addr[0],
            "destination.packets": 2,
            "destination.port": self.client_addr[1],
            "user.id": str(os.getuid()),
        })


class ConnectedUDP6TestCase:
    def __init__(self):
        pass

    def run(self):
        try:
            client, self.client_addr = socket_ipv6(socket.SOCK_DGRAM, socket.IPPROTO_UDP)
            server, self.server_addr = socket_ipv6(socket.SOCK_DGRAM, socket.IPPROTO_UDP)
            client.connect(self.server_addr)
            server.connect(self.client_addr)
            for i in range(5):
                server.send(bytes('Hello there {}'.format(i), "utf-8"))
                msg = client.recv(64)
            client.send(b'howdy')
            msg = server.recv(64)
            client.send(b'bye')
            msg = server.recv(64)
            client.close()
            server.close()
        finally:
            release_ipv6_address(self.server_addr)
            release_ipv6_address(self.client_addr)

    def expected(self):
        return HasEvent({
            "server.ip": self.client_addr[0],
            "server.port": self.client_addr[1],
            "source.bytes": Comparison(operator.gt, 250),
            "source.ip": self.server_addr[0],
            "source.packets": 5,
            "source.port": self.server_addr[1],
            "group.id": str(os.getgid()),
            "network.direction": "ingress",
            "network.packets": 7,
            "network.transport": "udp",
            "network.type": "ipv6",
            "process.pid": os.getpid(),
            "process.entity_id": Comparison(is_length, 16),
            "client.ip": self.server_addr[0],
            "client.port": self.server_addr[1],
            "destination.bytes": Comparison(operator.gt, 100),
            "destination.ip": self.client_addr[0],
            "destination.packets": 2,
            "destination.port": self.client_addr[1],
            "user.id": str(os.getuid()),
        })


class UDP6TestCase:
    def __init__(self):
        pass

    def run(self):
        try:
            client, self.client_addr = socket_ipv6(socket.SOCK_DGRAM, socket.IPPROTO_UDP)
            server, self.server_addr = socket_ipv6(socket.SOCK_DGRAM, socket.IPPROTO_UDP)
            for i in range(3):
                client.sendto(bytes('Hello there {}'.format(i), "utf-8"), self.server_addr)
                msg, _ = server.recvfrom(64)
            server.sendto(b'howdy', self.client_addr)
            msg, _ = client.recvfrom(64)
            client.close()
            server.close()
        finally:
            release_ipv6_address(self.server_addr)
            release_ipv6_address(self.client_addr)

    def expected(self):
        return HasEvent({
            "client.ip": self.client_addr[0],
            "client.port": self.client_addr[1],
            "destination.bytes": Comparison(operator.gt, 50),
            "destination.ip": self.server_addr[0],
            "destination.packets": 1,
            "destination.port": self.server_addr[1],
            "group.id": str(os.getgid()),
            "network.direction": "egress",
            "network.packets": 4,
            "network.transport": "udp",
            "network.type": "ipv6",
            "process.pid": os.getpid(),
            "process.entity_id": Comparison(is_length, 16),
            "server.ip": self.server_addr[0],
            "server.port": self.server_addr[1],
            "source.bytes": Comparison(operator.gt, 150),
            "source.ip": self.client_addr[0],
            "source.packets": 3,
            "source.port": self.client_addr[1],
            "user.id": str(os.getuid()),
        })


class MultiUDP4TestCase:
    def __init__(self):
        self.client_addr = None
        self.server_addr = [None] * 3

    def run(self):
        client, self.client_addr = socket_ipv4(socket.SOCK_DGRAM, socket.IPPROTO_UDP)
        for i in range(3):
            server, self.server_addr[i] = socket_ipv4(socket.SOCK_DGRAM, socket.IPPROTO_UDP)
            client.sendto(b'ping', self.server_addr[i])
            msg, _ = server.recvfrom(64)
            server.sendto(b'pong', self.client_addr)
            msg, _ = client.recvfrom(64)
            server.close()
        client.close()

    def expected(self):
        return HasEvent([
            {
                "client.ip": self.client_addr[0],
                "client.port": self.client_addr[1],
                "destination.bytes": Comparison(operator.gt, 30),
                "destination.ip": self.server_addr[0][0],
                "destination.packets": 1,
                "destination.port": self.server_addr[0][1],
                "group.id": str(os.getgid()),
                "network.direction": "egress",
                "network.packets": 2,
                "network.transport": "udp",
                "network.type": "ipv4",
                "process.pid": os.getpid(),
                "process.entity_id": Comparison(is_length, 16),
                "server.ip": self.server_addr[0][0],
                "server.port": self.server_addr[0][1],
                "source.bytes": Comparison(operator.gt, 30),
                "source.ip": self.client_addr[0],
                "source.packets": 1,
                "source.port": self.client_addr[1],
                "user.id": str(os.getuid()),
            },
            {
                "client.ip": self.client_addr[0],
                "client.port": self.client_addr[1],
                "destination.bytes": Comparison(operator.gt, 30),
                "destination.ip": self.server_addr[1][0],
                "destination.packets": 1,
                "destination.port": self.server_addr[1][1],
                "group.id": str(os.getgid()),
                "network.direction": "egress",
                "network.packets": 2,
                "network.transport": "udp",
                "network.type": "ipv4",
                "process.pid": os.getpid(),
                "process.entity_id": Comparison(is_length, 16),
                "server.ip": self.server_addr[1][0],
                "server.port": self.server_addr[1][1],
                "source.bytes": Comparison(operator.gt, 30),
                "source.ip": self.client_addr[0],
                "source.packets": 1,
                "source.port": self.client_addr[1],
                "user.id": str(os.getuid()),
            },
            {
                "client.ip": self.client_addr[0],
                "client.port": self.client_addr[1],
                "destination.bytes": Comparison(operator.gt, 30),
                "destination.ip": self.server_addr[2][0],
                "destination.packets": 1,
                "destination.port": self.server_addr[2][1],
                "group.id": str(os.getgid()),
                "network.direction": "egress",
                "network.packets": 2,
                "network.transport": "udp",
                "network.type": "ipv4",
                "process.pid": os.getpid(),
                "process.entity_id": Comparison(is_length, 16),
                "server.ip": self.server_addr[2][0],
                "server.port": self.server_addr[2][1],
                "source.bytes": Comparison(operator.gt, 30),
                "source.ip": self.client_addr[0],
                "source.packets": 1,
                "source.port": self.client_addr[1],
                "user.id": str(os.getuid()),
            },
        ])


class SocketFactory:

    def __init__(self, network, transport):
        self.network = network
        self.transport = transport
        if self.network == "ipv4":
            self.fn = socket_ipv4
        elif self.network == "ipv6":
            self.fn = socket_ipv6
        else:
            raise Exception("invalid network: " + self.network)
        if self.transport == "tcp":
            self.sock_type = socket.SOCK_STREAM
            self.sock_proto = socket.IPPROTO_TCP
        elif self.transport == "udp":
            self.sock_type = socket.SOCK_DGRAM
            self.sock_proto = socket.IPPROTO_UDP
        else:
            raise Exception("invalid transport: " + self.transport)

    def __call__(self, **kwargs):
        return self.fn(self.sock_type, self.sock_proto, **kwargs)


def transaction_udp(client, client_addr, server, server_addr, req, resp):
    client.sendto(req, server_addr)
    msg, _ = server.recvfrom(len(req))
    server.sendto(resp, client_addr)
    msg, _ = client.recvfrom(len(resp))


def transaction_tcp(client, client_addr, server, server_addr, req, resp):
    server.listen(8)
    client.connect(server_addr)
    accepted, _ = server.accept()
    client.send(req)
    accepted.recv(len(req))
    accepted.send(resp)
    client.recv(len(resp))
    accepted.close()


def transaction_udp_oneway(client, client_addr, server, server_addr, req, resp):
    client.sendto(req, server_addr)
    msg, _ = server.recvfrom(len(req))


class DNSTestCase:

    def __init__(self, enabled=True, delay_seconds=0, network="ipv4", transport="tcp", bidirectional=True):
        self.dns_enabled = enabled
        self.delay = delay_seconds
        self.socket_factory = SocketFactory(network, transport)
        self.transaction = transaction_tcp if transport == "tcp" else transaction_udp
        self.bidirectional = bidirectional
        if not self.bidirectional:
            assert transport == "udp"
            self.transaction = transaction_udp_oneway

    def run(self):
        A = "\x00\x01"
        AAAA = "\x00\x1c"
        q = A if self.socket_factory.network == "ipv4" else AAAA

        dns_factory = SocketFactory(self.socket_factory.network, "udp")
        dns_cli, self.dns_client_addr = dns_factory()
        dns_srv, self.dns_server_addr = dns_factory(port=53)
        client, self.client_addr = self.socket_factory()
        server, self.server_addr = self.socket_factory()

        raw_addr = ip_str_to_raw(self.server_addr[0])
        q_bytes = q.encode("utf-8")
        req = b"\x74\xba\x01\x00\x00\x01\x00\x00\x00\x00\x00\x00\x07elastic" \
              b"\x02co\x00" + q_bytes + b"\x00\x01"
        resp = b"\x74\xba\x81\x80\x00\x01\x00\x01\x00\x00\x00\x00\x07elastic" \
               b"\x02co\x00" + q_bytes + b"\x00\x01\xc0\x0c" + q_bytes + b"\x00\x01\x00\x00" \
               b"\x00\x9c" + struct.pack(">H", len(raw_addr)) + raw_addr

        transaction_udp(dns_cli, self.dns_client_addr,
                        dns_srv, self.dns_server_addr,
                        req, resp)
        dns_cli.close()
        dns_srv.close()
        time.sleep(self.delay)
        self.transaction(client, self.client_addr,
                         server, self.server_addr,
                         b"GET / HTTP/1.1\r\nHost: elastic.co\r\n\r\n",
                         b"HTTP/1.1 404 Not Found\r\n\r\n")
        client.close()
        server.close()

    def expected(self):

        if self.socket_factory.transport == "tcp":
            client_bytes = Comparison(operator.gt, 80)
            client_packets = Comparison(operator.gt, 2)
            server_bytes = Comparison(operator.gt, 2)
            server_packets = Comparison(operator.gt, 2)
            net_bytes = Comparison(operator.gt, 83)
            net_packets = Comparison(operator.gt, 5)
        else:
            client_bytes = Comparison(operator.gt, 5)
            client_packets = 1
            server_bytes = Comparison(operator.gt, 5) if self.bidirectional else 0
            server_packets = 0 + self.bidirectional
            net_bytes = Comparison(operator.gt, 5 + 6 * self.bidirectional)
            net_packets = 1 + self.bidirectional

        expected_events = [
            {
                "agent.type": "auditbeat",
                "client.bytes": Comparison(operator.gt, 30),
                "client.ip": self.dns_client_addr[0],
                "client.packets": 1,
                "client.port": self.dns_client_addr[1],
                "destination.bytes": Comparison(operator.gt, 30),
                "destination.ip": self.dns_server_addr[0],
                "destination.packets": 1,
                "destination.port": self.dns_server_addr[1],
                "event.action": "network_flow",
                "event.category": ["network", "network_traffic"],
                "event.dataset": "socket",
                "event.kind": "event",
                "event.module": "system",
                "network.bytes": Comparison(operator.gt, 60),
                "network.direction": "ingress",
                "network.packets": 2,
                "network.transport": "udp",
                "network.type": self.socket_factory.network,
                "process.pid": os.getpid(),
                "process.entity_id": Comparison(is_length, 16),
                "server.bytes": Comparison(operator.gt, 30),
                "server.ip": self.dns_server_addr[0],
                "server.packets": 1,
                "server.port": self.dns_server_addr[1],
                "source.bytes": Comparison(operator.gt, 30),
                "source.ip": self.dns_client_addr[0],
                "source.packets": 1,
                "source.port": self.dns_client_addr[1],
                "user.id": str(os.getuid()),
            }, {
                "agent.type": "auditbeat",
                "client.bytes": client_bytes,
                "client.ip": self.client_addr[0],
                "client.packets": client_packets,
                "client.port": self.client_addr[1],
                "destination.bytes": server_bytes,
                "destination.domain": "elastic.co",
                "destination.ip": self.server_addr[0],
                "destination.packets": server_packets,
                "destination.port": self.server_addr[1],
                "event.action": "network_flow",
                "event.category": ["network", "network_traffic"],
                "event.dataset": "socket",
                "event.kind": "event",
                "event.module": "system",
                "network.packets": net_bytes,
                "network.direction": "ingress",
                "network.packets": net_packets,
                "network.transport": self.socket_factory.transport,
                "network.type": self.socket_factory.network,
                "process.pid": os.getpid(),
                "process.entity_id": Comparison(is_length, 16),
                "server.bytes": server_bytes,
                "server.domain": "elastic.co",
                "server.ip": self.server_addr[0],
                "server.packets": server_packets,
                "server.port": self.server_addr[1],
                "service.type": "system",
                "source.bytes": client_bytes,
                "source.ip": self.client_addr[0],
                "source.packets": client_packets,
                "source.port": self.client_addr[1],
            },
        ]
        if not self.dns_enabled:
            for ev in expected_events:
                for k in [x for x in ev.keys() if x.endswith('.domain')]:
                    ev[k] = None

        return HasEvent(expected_events)


def socket_ipv4(type, proto, port=0):
    sock = socket.socket(socket.AF_INET, type, proto)
    sock.bind((random_address_ipv4(), port))
    return sock, sock.getsockname()


def random_address_ipv4():
    return '127.{}.{}.{}'.format(random.randint(0, 255), random.randint(0, 255), random.randint(1, 254))


def ip_str_to_raw(ip):
    return socket.inet_pton(socket.AF_INET6 if ':' in ip else socket.AF_INET, ip)


def socket_ipv6(type, proto, port=0):
    if not socket.has_ipv6:
        raise Exception('No IPv6 support!')
    addr = random_address_ipv6()
    rv = os.system('/sbin/ip -6 address add {}/128 dev lo'.format(addr))
    if rv != 0:
        raise Exception("add ip returned {}".format(rv))
    sock = socket.socket(socket.AF_INET6, type, proto)
    sock.bind((addr, port))
    return sock, sock.getsockname()


def release_ipv6_address(addr):
    if len(addr) == 0:
        return
    rv = os.system('/sbin/ip -6 address delete {}/128 dev lo'.format(addr[0]))
    if rv != 0:
        raise Exception("delete ip returned {}".format(rv))


def random_address_ipv6():
    return 'fdee:' + ':'.join(['{:x}'.format(random.randint(1, 65535)) for _ in range(7)])


class HasEvent:
    def __init__(self, expected):
        if isinstance(expected, dict):
            self.expected = [expected]
        elif isinstance(expected, list):
            self.expected = expected
        else:
            raise Exception("Wrong type")

    def __str__(self):
        return "the documents contain {}".format(
            ",\n".join(map(pretty_print_json, self.expected))
        )

    def match(self, output):
        documents = output
        expected = self.expected
        for (iexp, exp) in enumerate(expected):
            for (idoc, doc) in enumerate(documents):
                if all((k in doc and (doc[k] == v or callable(v) and v(doc[k]))) or (v is None and k not in doc)
                       for k, v in exp.items()):
                    break
            else:
                return False
            del documents[idoc]
        return True


class Comparison:
    def __init__(self, op, value):
        self.op = op
        self.value = value

    def __call__(self, n):
        return self.op(n, self.value)

    def to_json(self):
        return {
            "type": "comparison",
            "operator": str(self.op),
            "value": self.value,
        }
