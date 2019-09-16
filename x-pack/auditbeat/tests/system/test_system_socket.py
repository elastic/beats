import operator
import platform
import random
import json
import socket
import unittest
from auditbeat_xpack import *


def is_root():
    if 'geteuid' not in dir(os):
        return False
    euid = os.geteuid()
    return euid == 0


def is_version_below(version, target):
    t = map(int, target.split('.'))
    v = map(int, version.split('.'))
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
    f.write('0\n')
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
            except Exception, e:
                raise Exception('Auditbeat failed to start start'), None, sys.exc_info()[2]
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
        except Exception, e:
            raise Exception('No output received form Auditbeat'), None, sys.exc_info()[2]

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
        return map(lambda x: self.flatten_object(x, {}), self.read_output_json())


def pretty_print_json(d):
    return json.dumps(d, indent=3, default=lambda o: o.to_json(), sort_keys=True)


class TCP4TestCase:
    def __init__(self):
        pass

    def run(self):
        client, self.client_addr = socket_ipv4(socket.SOCK_STREAM, socket.IPPROTO_TCP)
        server, self.server_addr = socket_ipv4(socket.SOCK_STREAM, socket.IPPROTO_TCP)
        server.listen(8)
        client.connect(self.server_addr)
        acc, _ = server.accept()
        acc.send('Hello there\n')
        msg = client.recv(64)
        client.send('"{}" what\n'.format(msg))
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
            "network.direction": "outbound",
            "group.id": str(os.getgid()),
            "user.id": str(os.getuid()),
            "process.pid": os.getpid(),
        })


class UDP4TestCase:
    def __init__(self):
        pass

    def run(self):
        client, self.client_addr = socket_ipv4(socket.SOCK_DGRAM, socket.IPPROTO_UDP)
        server, self.server_addr = socket_ipv4(socket.SOCK_DGRAM, socket.IPPROTO_UDP)
        for i in range(3):
            client.sendto('Hello there {}'.format(i), self.server_addr)
            msg, _ = server.recvfrom(64)
        server.sendto('howdy', self.client_addr)
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
            "network.direction": "outbound",
            "network.packets": 4,
            "network.transport": "udp",
            "network.type": "ipv4",
            "process.pid": os.getpid(),
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
            server.send('Hello there {}'.format(i))
            msg = client.recv(64)
        client.send('howdy')
        msg = server.recv(64)
        client.send('bye')
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
            "network.direction": "inbound",
            "network.packets": 7,
            "network.transport": "udp",
            "network.type": "ipv4",
            "process.pid": os.getpid(),
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
                server.send('Hello there {}'.format(i))
                msg = client.recv(64)
            client.send('howdy')
            msg = server.recv(64)
            client.send('bye')
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
            "network.direction": "inbound",
            "network.packets": 7,
            "network.transport": "udp",
            "network.type": "ipv6",
            "process.pid": os.getpid(),
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
                client.sendto('Hello there {}'.format(i), self.server_addr)
                msg, _ = server.recvfrom(64)
            server.sendto('howdy', self.client_addr)
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
            "network.direction": "outbound",
            "network.packets": 4,
            "network.transport": "udp",
            "network.type": "ipv6",
            "process.pid": os.getpid(),
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
            client.sendto('ping', self.server_addr[i])
            msg, _ = server.recvfrom(64)
            server.sendto('pong', self.client_addr)
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
                "network.direction": "outbound",
                "network.packets": 2,
                "network.transport": "udp",
                "network.type": "ipv4",
                "process.pid": os.getpid(),
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
                "network.direction": "outbound",
                "network.packets": 2,
                "network.transport": "udp",
                "network.type": "ipv4",
                "process.pid": os.getpid(),
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
                "network.direction": "outbound",
                "network.packets": 2,
                "network.transport": "udp",
                "network.type": "ipv4",
                "process.pid": os.getpid(),
                "server.ip": self.server_addr[2][0],
                "server.port": self.server_addr[2][1],
                "source.bytes": Comparison(operator.gt, 30),
                "source.ip": self.client_addr[0],
                "source.packets": 1,
                "source.port": self.client_addr[1],
                "user.id": str(os.getuid()),
            },
        ])


def socket_ipv4(type, proto):
    sock = socket.socket(socket.AF_INET, type, proto)
    sock.bind((random_address_ipv4(), 0))
    return sock, sock.getsockname()


def random_address_ipv4():
    return '127.{}.{}.{}'.format(random.randint(0, 255), random.randint(0, 255), random.randint(1, 254))


def socket_ipv6(type, proto):
    if not socket.has_ipv6:
        raise Exception('No IPv6 support!')
    addr = random_address_ipv6()
    rv = os.system('/sbin/ip -6 address add {}/128 dev lo'.format(addr))
    if rv != 0:
        raise Exception("add ip returned {}".format(rv))
    sock = socket.socket(socket.AF_INET6, type, proto)
    sock.bind((addr, 0))
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
                if all(k in doc and (doc[k] == v or callable(v) and v(doc[k]))
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
