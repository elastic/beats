# This file is derived from test_tcp_tls.py and adapted to specifically
# test that TLS 1.3 is active only when the correct environment variable
# is set (BEATS_TLS13=1).

from filebeat import BaseTest
import socket
import ssl
import unittest
from os import path
from nose.tools import raises, assert_raises

NUMBER_OF_EVENTS = 2

CURRENT_PATH = path.dirname(__file__)
CERTPATH = path.abspath(path.join(CURRENT_PATH, "config/certificates"))


# Self signed certificate used without mutual and failing scenario.
CERTIFICATE1 = path.join(CERTPATH, "beats1.crt")
KEY1 = path.join(CERTPATH, "beats1.key")

CERTIFICATE2 = path.join(CERTPATH, "beats2.crt")
KEY2 = path.join(CERTPATH, "beats2.key")


# Valid self signed certificate used for mutual auth.
CACERT = path.join(CERTPATH, "cacert.crt")

CLIENT1 = path.join(CERTPATH, "client1.crt")
CLIENTKEY1 = path.join(CERTPATH, "client1.key")

CLIENT2 = path.join(CERTPATH, "client2.crt")
CLIENTKEY2 = path.join(CERTPATH, "client2.key")


class Test(BaseTest):
    """
    Test filebeat TCP input with TLS 1.3
    """

    def test_tcp_over_tls13_mutual_auth_succeed(self):
        """
        Test filebeat TCP with TLS when enforcing client auth with good client certificates.
        """
        # TODO: add   ssl.supported_protocols: [TLSv1.3]
        input_raw = """
- type: tcp
  host: "{host}:{port}"
  enabled: true
  ssl.certificate_authorities: {cacert}
  ssl.certificate: {certificate}
  ssl.key: {key}
  ssl.client_authentication: required
"""
        config = {
            "host": "127.0.0.1",
            "port": 8080,
            "cacert": CACERT,
            "certificate": CLIENT1,
            "key": CLIENTKEY1,
        }

        input_raw = input_raw.format(**config)

        self.render_config_template(
            input_raw=input_raw,
            inputs=False,
        )

        filebeat = self.start_beat()

        self.wait_until(lambda: self.log_contains(
            "Started listening for TCP connection"))

        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)

        context = ssl.create_default_context(ssl.Purpose.CLIENT_AUTH)
        context.verify_mode = ssl.CERT_REQUIRED
        context.load_verify_locations(CACERT)
        context.load_cert_chain(certfile=CLIENT2, keyfile=CLIENTKEY2)

        tls = context.wrap_socket(sock, server_side=False)

        tls.connect((config.get('host'), config.get('port')))

        for n in range(0, NUMBER_OF_EVENTS):
            tls.send("Hello World: " + str(n) + "\n")

        self.wait_until(lambda: self.output_count(
            lambda x: x >= NUMBER_OF_EVENTS))

        filebeat.check_kill_and_wait()

        output = self.read_output()

        self.assert_output(output)

        sock.close()

    def assert_output(self, output):
        assert len(output) == 2
        assert output[0]["input.type"] == "tcp"
