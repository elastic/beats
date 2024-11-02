from filebeat import BaseTest
import os
import socket
import unittest


class Test(BaseTest):

    @unittest.skipIf(os.name == 'nt', 'flaky test https://github.com/elastic/beats/issues/22809')
    def test_udp(self):
        """
        Test UDP input with it binding to 127.0.0.1 (default).
        """
        self.send_events_with_bind()

    @unittest.skipIf(os.name == 'nt', 'flaky test https://github.com/elastic/beats/issues/22809')
    def test_udp_with_wildcard_address(self):
        """
        Test UDP input with it binding to the wildcard address 0.0.0.0.
        """
        self.send_events_with_bind(bind="0.0.0.0")

    def send_events_with_bind(self, bind="127.0.0.1"):

        host = "127.0.0.1"
        port = 8080
        input_raw = """
- type: udp
  host: "{}:{}"
  enabled: true
"""

        input_raw = input_raw.format(bind, port)

        self.render_config_template(
            input_raw=input_raw,
            inputs=False,
        )

        filebeat = self.start_beat()

        self.wait_until(lambda: self.log_contains("Started listening for UDP connection"))

        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)  # UDP

        for n in range(0, 2):
            sock.sendto(b"Hello World: " + n.to_bytes(2, "big"), (host, port))

        self.wait_until(lambda: self.output_count(lambda x: x >= 2))

        filebeat.check_kill_and_wait()

        output = self.read_output()

        assert len(output) == 2
        assert output[0]["input.type"] == "udp"
