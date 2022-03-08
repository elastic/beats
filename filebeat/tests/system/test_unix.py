import os
import platform
import socket
import tempfile
import unittest
from filebeat import BaseTest

# AF_UNIX support in python isn't available until
# Python 3.9, see https://bugs.python.org/issue33408


@unittest.skipIf(not hasattr(socket, 'AF_UNIX'), "No Windows AF_UNIX support before Python 3.9")
@unittest.skipIf(platform.system() == 'Darwin', 'Flaky test: https://github.com/elastic/beats/issues/20941')
class Test(BaseTest):
    """
    Test filebeat UNIX input
    """

    def test_unix_with_newline_delimiter(self):
        """
        Test UNIX input with a new line delimiter
        """
        self.send_events_with_delimiter("\n")

    def test_unix_with_custom_char_delimiter(self):
        """
        Test UNIX input with a custom single char delimiter
        """
        self.send_events_with_delimiter(";")

    def test_unix_with_custom_word_delimiter(self):
        """
        Test UNIX input with a custom single char delimiter
        """
        self.send_events_with_delimiter("<END>")

    def send_events_with_delimiter(self, delimiter):
        # we create the socket in a temporary directory because
        # go will fail to create a unix socket if the path length
        # is longer than 108 characters. See https://github.com/golang/go/issues/6895
        with tempfile.TemporaryDirectory() as tempdir:
            path = os.path.join(tempdir, "filebeat.sock")
            input_raw = """
- type: unix
  path: {}
  enabled: true
"""

            # Use default of \n and stripping \r
            if delimiter != "":
                input_raw += "\n  line_delimiter: {}".format(delimiter)

            input_raw = input_raw.format(path)

            self.render_config_template(
                input_raw=input_raw,
                inputs=False,
            )

            filebeat = self.start_beat()

            self.wait_until(lambda: self.log_contains("Start accepting connections"))

            sock = send_stream_socket(path, delimiter)

            self.wait_until(lambda: self.output_count(lambda x: x >= 2))

            filebeat.check_kill_and_wait()

            output = self.read_output()

            assert len(output) == 2
            assert output[0]["input.type"] == "unix"

            sock.close()

    def test_unix_datagram_socket(self):
        # we create the socket in a temporary directory because
        # go will fail to create a unix socket if the path length
        # is longer than 108 characters. See https://github.com/golang/go/issues/6895
        with tempfile.TemporaryDirectory() as tempdir:
            path = os.path.join(tempdir, "filebeat.sock")
            input_raw = """
- type: unix
  path: {}
  enabled: true
  socket_type: datagram
"""

            input_raw = input_raw.format(path)

            self.render_config_template(
                input_raw=input_raw,
                inputs=False,
            )

            filebeat = self.start_beat()

            self.wait_until(lambda: self.log_contains("Started listening for UNIX connection"))

            sock = send_datagram_socket(path)

            self.wait_until(lambda: self.output_count(lambda x: x >= 2))

            filebeat.check_kill_and_wait()

            output = self.read_output()

            assert len(output) == 2
            assert output[0]["message"] == "Hello World: 0;"
            assert output[0]["input.type"] == "unix"

            sock.close()


def send_stream_socket(path, delimiter):
    sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
    sock.connect(path)

    for n in range(0, 2):
        sock.send(bytes("Hello World: " + str(n) + delimiter, "utf-8"))

    return sock


def send_datagram_socket(path):
    sock = socket.socket(socket.AF_UNIX, socket.SOCK_DGRAM)
    sock.connect(path)

    for n in range(0, 2):
        sock.sendto(bytes("Hello World: " + str(n) + ";", "utf-8"), path)

    return sock
