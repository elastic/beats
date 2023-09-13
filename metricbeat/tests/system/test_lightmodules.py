import http.server
import metricbeat
import os
import os.path
import platform
import shutil
import sys
import threading
import unittest
import json

from contextlib import contextmanager


class Test(metricbeat.BaseTest):
    @unittest.skipIf(platform.platform().startswith("Windows-10"),
                     "flaky test: https://github.com/elastic/beats/issues/26181")
    def test_processors(self):
        shutil.copytree(
            os.path.join(self.beat_path, "mb/testing/testdata/lightmodules"),
            os.path.join(self.working_dir, "module"),
        )

        with http_test_server() as server:
            self.render_config_template(modules=[{
                "name": "test",
                "metricsets": ["json"],
                "namespace": "test",
                # Hard-coding 'localhost' because hostname in server.server_name doesn't always work.
                "hosts": [f"localhost:{server.server_port}"],
            }])

            proc = self.start_beat()

            self.wait_until(lambda: self.output_lines() > 0)
            proc.check_kill_and_wait()

        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            self.assertEqual(evt["fields"]["test"], "fromprocessor")


@contextmanager
def http_test_server():
    server = http.server.HTTPServer(('localhost', 0), HTTPHandlerForTest)
    child = threading.Thread(target=server.serve_forever)
    child.start()
    yield server
    server.shutdown()
    child.join()


class HTTPHandlerForTest(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header("Content-type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps({"foo": "bar"}).encode("utf-8"))
