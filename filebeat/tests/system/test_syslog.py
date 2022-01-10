from filebeat import BaseTest
import socket
import os
import tempfile
import unittest


class Test(BaseTest):
    """
    Test filebeat with the syslog input
    """

    def test_syslog_with_tcp(self):
        """
        Test syslog input with events from TCP.
        """
        host = "127.0.0.1"
        port = 8080
        input_raw = """
- type: syslog
  protocol:
    tcp:
        host: "{}:{}"
"""

        input_raw = input_raw.format(host, port)
        self.render_config_template(
            input_raw=input_raw,
            inputs=False,
        )

        filebeat = self.start_beat()

        self.wait_until(lambda: self.log_contains("Start accepting connections"))

        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)  # TCP
        sock.connect((host, port))

        for n in range(0, 2):
            m = "<13>Oct 11 22:14:15 wopr.mymachine.co postfix/smtpd[2000]:" \
                " 'su root' failed for lonvick on /dev/pts/8 {}\n"
            m = m.format(n)
            sock.send(m.encode("utf-8"))

        self.wait_until(lambda: self.output_count(lambda x: x >= 2))

        filebeat.check_kill_and_wait()

        output = self.read_output()

        assert len(output) == 2
        self.assert_syslog(output[0])
        sock.close()

    def test_syslog_with_tcp_invalid_message(self):
        """
        Test syslog input with invalid events from TCP.
        """
        host = "127.0.0.1"
        port = 8080
        input_raw = """
- type: syslog
  protocol:
    tcp:
        host: "{}:{}"
"""

        input_raw = input_raw.format(host, port)
        self.render_config_template(
            input_raw=input_raw,
            inputs=False,
        )

        filebeat = self.start_beat()

        self.wait_until(lambda: self.log_contains("Start accepting connections"))

        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)  # TCP
        sock.connect((host, port))

        for n in range(0, 2):
            sock.send("invalid\n".encode("utf-8"))

        self.wait_until(lambda: self.output_count(lambda x: x >= 2))

        filebeat.check_kill_and_wait()

        output = self.read_output()

        assert len(output) == 2
        assert output[0]["message"] == "invalid"
        assert len(output[0]["log.source.address"]) > 0
        sock.close()

    def test_syslog_with_udp(self):
        """
        Test syslog input with events from TCP.
        """
        host = "127.0.0.1"
        port = 8080
        input_raw = """
- type: syslog
  protocol:
    udp:
        host: "{}:{}"
"""

        input_raw = input_raw.format(host, port)
        self.render_config_template(
            input_raw=input_raw,
            inputs=False,
        )

        filebeat = self.start_beat()

        self.wait_until(lambda: self.log_contains("Started listening for UDP connection"))

        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)  # TCP

        for n in range(0, 50):
            m = "<13>Oct 11 22:14:15 wopr.mymachine.co postfix/smtpd[2000]:" \
                " 'su root' failed for lonvick on /dev/pts/8 {}\n"
            m = m.format(n)
            sock.sendto(m.encode("utf-8"), (host, port))

        self.wait_until(lambda: self.output_count(lambda x: x >= 1))
        filebeat.check_kill_and_wait()
        sock.close()

        output = self.read_output()
        self.assert_syslog(output[0])

    # AF_UNIX support in python isn't available until
    # Python 3.9, see https://bugs.python.org/issue33408
    @unittest.skipIf(not hasattr(socket, 'AF_UNIX'), "No Windows AF_UNIX support before Python 3.9")
    def test_syslog_with_unix_stream(self):
        """
        Test syslog input with events from UNIX stream.
        """

        self.run_filebeat_and_send_using_socket("stream", send_stream_socket)

    # AF_UNIX support in python isn't available until
    # Python 3.9, see https://bugs.python.org/issue33408
    @unittest.skipIf(not hasattr(socket, 'AF_UNIX'), "No Windows AF_UNIX support before Python 3.9")
    def test_syslog_with_unix_datagram(self):
        """
        Test syslog input with events from UNIX stream.
        """

        self.run_filebeat_and_send_using_socket("datagram", send_datagram_socket)

    def run_filebeat_and_send_using_socket(self, socket_type, send_over_socket):
        # we create the socket in a temporary directory because
        # go will fail to create a unix socket if the path length
        # is longer than 108 characters. See https://github.com/golang/go/issues/6895

        with tempfile.TemporaryDirectory() as tempdir:
            path = os.path.join(tempdir, "filebeat.sock")
            input_raw = """
- type: syslog
  protocol:
    unix:
        path: {}
        socket_type: {}
"""

            input_raw = input_raw.format(path, socket_type)
            self.render_config_template(
                input_raw=input_raw,
                inputs=False,
            )

            filebeat = self.start_beat()

            if socket_type == "stream":
                self.wait_until(lambda: self.log_contains("Start accepting connections"))
            else:
                self.wait_until(lambda: self.log_contains("Started listening"))

            sock = send_over_socket(path,
                                    "<13>Oct 11 22:14:15 wopr.mymachine.co postfix/smtpd[2000]:"
                                    " 'su root' failed for lonvick on /dev/pts/8 {}\n")

            self.wait_until(lambda: self.output_count(lambda x: x >= 2))

            filebeat.check_kill_and_wait()

            output = self.read_output()

            assert len(output) == 2
            self.assert_syslog(output[0], False)

            sock.close()

    # AF_UNIX support in python isn't available until
    # Python 3.9, see https://bugs.python.org/issue33408

    @unittest.skipIf(not hasattr(socket, 'AF_UNIX'), "No Windows AF_UNIX support before Python 3.9")
    def test_syslog_with_unix_stream_invalid_message(self):
        """
        Test syslog input with invalid events from UNIX.
        """

        self.run_filebeat_and_send_invalid_message_using_socket("stream", send_stream_socket)

    # AF_UNIX support in python isn't available until
    # Python 3.9, see https://bugs.python.org/issue33408
    @unittest.skipIf(not hasattr(socket, 'AF_UNIX'), "No Windows AF_UNIX support before Python 3.9")
    def test_syslog_with_unix_datagram_invalid_message(self):
        """
        Test syslog input with invalid events from UNIX.
        """

        self.run_filebeat_and_send_invalid_message_using_socket("datagram", send_datagram_socket)

    def run_filebeat_and_send_invalid_message_using_socket(self, socket_type, send_over_socket):
        # we create the socket in a temporary directory because
        # go will fail to create a unix socket if the path length
        # is longer than 108 characters. See https://github.com/golang/go/issues/6895
        with tempfile.TemporaryDirectory() as tempdir:
            path = os.path.join(tempdir, "filebeat.sock")
            input_raw = """
- type: syslog
  protocol:
    unix:
        path: {}
        socket_type: {}
"""

            input_raw = input_raw.format(path, socket_type)
            self.render_config_template(
                input_raw=input_raw,
                inputs=False,
            )

            filebeat = self.start_beat()

            if socket_type == "stream":
                self.wait_until(lambda: self.log_contains("Start accepting connections"))
            else:
                self.wait_until(lambda: self.log_contains("Started listening"))

            sock = send_over_socket(path, "invalid\n")

            self.wait_until(lambda: self.output_count(lambda x: x >= 2))

            filebeat.check_kill_and_wait()

            output = self.read_output()

            assert len(output) == 2
            expected_message = "invalid"
            if socket_type == "datagram":
                expected_message += "\n"
            assert output[0]["message"] == expected_message
            sock.close()

    def assert_syslog(self, syslog, has_address=True):
        assert syslog["event.severity"] == 5
        assert syslog["hostname"] == "wopr.mymachine.co"
        assert syslog["input.type"] == "syslog"
        assert syslog["message"].startswith("'su root' failed for lonvick on /dev/pts/8")
        assert syslog["process.pid"] == 2000
        assert syslog["process.program"] == "postfix/smtpd"
        assert syslog["syslog.facility"] == 1
        assert syslog["syslog.priority"] == 13
        assert syslog["syslog.severity_label"] == "Notice"
        assert syslog["syslog.facility_label"] == "user-level"
        if has_address:
            assert len(syslog["log.source.address"]) > 0


def send_stream_socket(path, message):
    sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)

    sock.connect(path)

    for n in range(0, 2):
        message = message.format(n)
        sock.send(message.encode("utf-8"))

    return sock


def send_datagram_socket(path, message):
    sock = socket.socket(socket.AF_UNIX, socket.SOCK_DGRAM)

    for n in range(0, 2):
        message = message.format(n)
        sock.sendto(message.encode("utf-8"), path)

    return sock
