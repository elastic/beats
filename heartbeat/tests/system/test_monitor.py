from heartbeat import BaseTest
import BaseHTTPServer
import threading
from parameterized import parameterized
import os
from nose.plugins.skip import SkipTest


class Test(BaseTest):

    @parameterized.expand([
        "200", "404"
    ])
    def test_http(self, status_code):
        """
        Test http server
        """
        status_code = int(status_code)
        server = self.start_server("hello world", status_code)

        self.render_config_template(
            monitors=[{
                "type": "http",
                "urls": ["http://localhost:8185"],
            }],
        )

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("heartbeat is running"))

        self.wait_until(
            lambda: self.output_has(lines=1))

        proc.check_kill_and_wait()

        server.shutdown()
        output = self.read_output()
        assert status_code == output[0]["http.response.status_code"]

        if os.name == "nt":
            # Currently skipped on Windows as fields.yml not generated
            raise SkipTest
        self.assert_fields_are_documented(output[0])

    @parameterized.expand([
        ("8185", "up"),
        ("8186", "down"),
    ])
    def test_tcp(self, port, status):
        """
        Test tcp server
        """
        server = self.start_server("hello world", 200)
        self.render_config_template(
            monitors=[{
                "type": "tcp",
                "hosts": ["localhost:" + port],
            }],
        )

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("heartbeat is running"))

        self.wait_until(
            lambda: self.output_has(lines=1))

        proc.check_kill_and_wait()

        server.shutdown()

        output = self.read_output()
        assert status == output[0]["monitor.status"]
        if os.name == "nt":
            # Currently skipped on Windows as fields.yml not generated
            raise SkipTest
        self.assert_fields_are_documented(output[0])

    def start_server(self, content, status_code):
        class HTTPHandler(BaseHTTPServer.BaseHTTPRequestHandler):
            def do_GET(self):
                self.send_response(status_code)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                self.wfile.write(content)

        server = BaseHTTPServer.HTTPServer(('localhost', 8185), HTTPHandler)

        thread = threading.Thread(target=server.serve_forever)
        thread.start()

        return server
