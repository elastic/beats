import os
import sys
import BaseHTTPServer
import threading
import nose.tools

sys.path.append(os.path.join(os.path.dirname(
    __file__), '../../../libbeat/tests/system'))

from beat.beat import TestCase
from time import sleep


class BaseTest(TestCase):

    @classmethod
    def setUpClass(self):
        self.beat_name = "heartbeat"
        self.beat_path = os.path.abspath(
            os.path.join(os.path.dirname(__file__), "../../"))
        super(BaseTest, self).setUpClass()

    def start_server(self, content, status_code, **kwargs):
        class HTTPHandler(BaseHTTPServer.BaseHTTPRequestHandler):
            def do_GET(self):
                self.send_response(status_code)
                self.send_header('Content-Type', 'application/json')
                self.end_headers()
                if "write_delay" in kwargs:
                    sleep(float(kwargs["write_delay"]))

                self.wfile.write(content)

        server = BaseHTTPServer.HTTPServer(('localhost', 0), HTTPHandler)

        thread = threading.Thread(target=server.serve_forever)
        thread.start()

        return server

    @staticmethod
    def http_cfg(id, url):
        return """
- type: http
  id: "{id}"
  schedule: "@every 1s"
  timeout: 3s
  urls: ["{url}"]
        """[1:-1].format(id=id, url=url)

    @staticmethod
    def tcp_cfg(*hosts):
        host_str = ", ".join('"' + host + '"' for host in hosts)
        return """
- type: tcp
  schedule: "@every 1s"
  timeout: 3s
  hosts: [{host_str}]
        """[1:-1].format(host_str=host_str)

    def last_output_line(self):
        return self.read_output()[-1]

    def write_dyn_config(self, filename, cfg):
        with open(self.monitors_dir() + filename, 'w') as f:
            f.write(cfg)

    def monitors_dir(self):
        return self.working_dir + "/monitors.d/"

    def assert_last_status(self, status):
        nose.tools.eq_(self.last_output_line()["monitor.status"], status)

    def setup_dynamic(self, extra_beat_args=[]):
        os.mkdir(self.monitors_dir())
        self.render_config_template(
            reload=True,
            reload_path=self.monitors_dir() + "*.yml",
            flush_min_events=1,
        )

        self.proc = self.start_beat(extra_args=extra_beat_args)
