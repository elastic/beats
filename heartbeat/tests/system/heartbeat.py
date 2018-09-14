import os
import sys
import BaseHTTPServer
import threading

sys.path.append(os.path.join(os.path.dirname(
    __file__), '../../../libbeat/tests/system'))

from beat.beat import TestCase


class BaseTest(TestCase):

    @classmethod
    def setUpClass(self):
        self.beat_name = "heartbeat"
        self.beat_path = os.path.abspath(
            os.path.join(os.path.dirname(__file__), "../../"))
        super(BaseTest, self).setUpClass()

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
