from heartbeat import BaseTest
import BaseHTTPServer
import threading
from parameterized import parameterized


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
                "urls": ["http://localhost:8181"],
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

    @parameterized.expand([
        ("8181", "up"),
        ("8182", "down"),
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

    def start_server(self, content, status_code):
        class HTTPHandler(BaseHTTPServer.BaseHTTPRequestHandler):
            def do_GET(self):
                self.send_response(status_code)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                self.wfile.write(content)

        server = BaseHTTPServer.HTTPServer(('localhost', 8181), HTTPHandler)

        thread = threading.Thread(target=server.serve_forever)
        thread.start()

        return server
