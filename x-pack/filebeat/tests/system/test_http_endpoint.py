import requests
import sys
import os

sys.path.append(os.path.join(os.path.dirname(__file__), '../../../../filebeat/tests/system/'))
from filebeat import BaseTest


class Test(BaseTest):
    """
    Test filebeat with the http_endpoint input
    """

    def test_http_endpoint_without_ssl(self):
        """
        Test http_endpoint input with HTTP events.
        """
        host = "127.0.0.1"
        port = 8081
        input_raw = """
- type: http_endpoint
  enabled: true
    listen_address: {}
    listen_port: {}
"""

        input_raw = input_raw.format(host, port)
        self.render_config_template(
            input_raw=input_raw,
            inputs=False,
        )

        filebeat = self.start_beat()

        self.wait_until(lambda: self.log_contains("Starting HTTP server on 127.0.0.1:8081"))