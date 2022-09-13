import os
import unittest

from heartbeat import BaseTest
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

        self.render_http_config(
            ["localhost:{}".format(server.server_port)])

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
            raise unittest.SkipTest
        self.assert_fields_are_documented(output[0])

    @parameterized.expand([
        "200", "404"
    ])
    def test_http_with_hosts_config(self, status_code):
        """
        Test http server
        """
        status_code = int(status_code)
        server = self.start_server("hello world", status_code)

        self.render_http_config_with_hosts(
            ["localhost:{}".format(server.server_port)])

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
            raise unittest.SkipTest
        self.assert_fields_are_documented(output[0])

    @parameterized.expand([
        # enable_options_method
        (lambda enable_options: True, 200),
        (lambda enable_options: False, 501),
    ])
    def test_http_check_with_options_method(self, enable_options, status_code):
        """
        Test http server if it supports OPTIONS method check
        """
        # get enable_options value from parameterized decorator
        enable_options = enable_options(enable_options)
        status_code = int(status_code)
        server = self.start_server("hello world", status_code,
                                   enable_options_method=enable_options)

        self.render_http_config_with_options_method(
            ["localhost:{}".format(server.server_port)])

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("heartbeat is running"))

        self.wait_until(
            lambda: self.output_has(lines=1))

        proc.check_kill_and_wait()

        server.shutdown()
        output = self.read_output()
        assert status_code == output[0]["http.response.status_code"]
        if enable_options:
            # make sure OPTIONS is in the allowed methods from the response
            assert -1 != output[0]["http.response.headers.Access-Control-Allow-Methods"].find("OPTIONS")

        if os.name == "nt":
            # Currently skipped on Windows as fields.yml not generated
            raise unittest.SkipTest
        self.assert_fields_are_documented(output[0])

    def test_http_delayed(self):
        """
        Ensure that the HTTP monitor consumes the whole body.
        We do this by ensuring that a slow HTTP body write's time is reflected
        in the beats metrics.
        """
        try:
            delay = 1.0
            server = self.start_server("sloooow body", 200, write_delay=delay)

            self.render_http_config(
                ["http://localhost:{}".format(server.server_port)])

            try:
                proc = self.start_beat()
                self.wait_until(lambda: self.output_has(lines=1))
                self.assertGreaterEqual(
                    self.last_output_line()['http.rtt.total.us'], delay)
            finally:
                proc.check_kill_and_wait()
        finally:
            server.shutdown()

    @parameterized.expand([
        (lambda server: "localhost:{}".format(server.server_port), "up"),
        # This IP is reserved in IPv4
        (lambda server: "203.0.113.1:1233", "down"),
    ])
    def test_tcp(self, url, status):
        """
        Test tcp server
        """
        server = self.start_server("hello world", 200)
        try:
            self.render_config_template(
                monitors=[{
                    "type": "tcp",
                    "hosts": [url(server)],
                    "timeout": "3s"
                }],
            )

            proc = self.start_beat()
            try:
                self.wait_until(lambda: self.log_contains(
                    "heartbeat is running"))

                self.wait_until(
                    lambda: self.output_has(lines=1))
            finally:
                proc.check_kill_and_wait()

            output = self.read_output()
            self.assert_last_status(status)
            if os.name == "nt":
                # Currently skipped on Windows as fields.yml not generated
                raise unittest.SkipTest
            self.assert_fields_are_documented(output[0])
        finally:
            server.shutdown()

    def render_http_config(self, urls):
        self.render_config_template(
            monitors=[{
                "type": "http",
                "urls": urls,
            }]
        )

    def render_http_config_with_hosts(self, urls):
        self.render_config_template(
            monitors=[{
                "type": "http",
                "hosts": urls,
            }]
        )

    def render_http_config_with_options_method(self, urls):
        self.render_config_template(
            monitors=[{
                "type": "http",
                "hosts": urls,
                "check_request_method": "OPTIONS",
            }]
        )
