import platform
import random
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

    def with_runner(self, test):
        self.render_config_template(modules=[{
            "name": "system",
            "datasets": ["socket"],
            "extras": {
                "socket.flow_inactive_timeout": "2s",
                "socket.flow_termination_timeout": "5s",
                "socket.development_mode": "true",
            }
        }])
        proc = self.start_beat()
        try:
            self.wait_until(lambda: self.log_contains('system/socket dataset is running.'),
                            max_timeout=60)
            self.execute(test)
        except Exception:
            proc.check_kill_and_wait()
            raise
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

        self.wait_until(lambda: self.output_lines() > 0)
        self.wait_until(lambda: test.completed(map(lambda x: self.flatten_object(x, {}), self.read_output_json())))


class TCP4TestCase:
    def __init__(self):
        pass

    def run(self):
        client, self.client_addr = socket_ipv4(socket.SOCK_STREAM, 0)
        server, self.server_addr = socket_ipv4(socket.SOCK_STREAM, 0)
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

    def completed(self, output):
        return has_event(output, {
            "source.ip": self.client_addr[0],
            "source.port": self.client_addr[1],
            "source.bytes": lambda x: x > 20,
            "source.packets": lambda x: x > 4,
            "client.ip": self.client_addr[0],
            "client.port": self.client_addr[1],
            "destination.ip": self.server_addr[0],
            "destination.port": self.server_addr[1],
            "destination.bytes": lambda x: x > 20,
            "destination.packets": lambda x: x > 4,
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
        client, self.client_addr = socket_ipv4(socket.SOCK_DGRAM, 0)
        server, self.server_addr = socket_ipv4(socket.SOCK_DGRAM, 0)
        for i in range(3):
            client.sendto('Hello there {}'.format(i), self.server_addr)
            msg, _ = server.recvfrom(64)
        server.sendto('howdy', self.client_addr)
        msg, _ = client.recvfrom(64)
        client.close()
        server.close()

    def completed(self, output):
        return has_event(output, {
            "client.ip": self.client_addr[0],
            "client.port": self.client_addr[1],
            "destination.bytes": lambda x: x > 0,
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
            "source.bytes": lambda x: x >= 40,
            "source.ip": self.client_addr[0],
            "source.packets": 3,
            "source.port": self.client_addr[1],
            "user.id": str(os.getuid()),
        })


class ConnectedUDP4TestCase:
    def __init__(self):
        pass

    def run(self):
        client, self.client_addr = socket_ipv4(socket.SOCK_DGRAM, 0)
        server, self.server_addr = socket_ipv4(socket.SOCK_DGRAM, 0)
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

    def completed(self, output):
        return has_event(output, {
            "server.ip": self.client_addr[0],
            "server.port": self.client_addr[1],
            "source.bytes": lambda x: x > 50,
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
            "destination.bytes": lambda x: x >= 0,
            "destination.ip": self.client_addr[0],
            "destination.packets": 2,
            "destination.port": self.client_addr[1],
            "user.id": str(os.getuid()),
        })


def socket_ipv4(type, proto):
    sock = socket.socket(socket.AF_INET, type, proto)
    sock.bind((random_address_ipv4(), 0))
    return sock, sock.getsockname()


def random_address_ipv4():
    return '127.{}.{}.{}'.format(random.randint(0, 255), random.randint(0, 255), random.randint(1, 254))


def has_event(list, expected):
    # True if any of the events in list contains all the keys in expected.
    return any(all(k in obj and (obj[k] == v or callable(v) and v(obj[k])) for k, v in expected.items()) for obj in list)
