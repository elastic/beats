from heartbeat import BaseTest
from parameterized import parameterized
import os
from nose.plugins.skip import SkipTest
import nose.tools


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
            ["http://localhost:{}".format(server.server_port)])

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
                nose.tools.assert_greater_equal(
                    self.last_output_line()['http.rtt.total.us'], delay)
            finally:
                proc.check_kill_and_wait()
        finally:
            server.shutdown()

    @parameterized.expand([
        ("up", '{"foo": {"baz": "bar"}}'),
        ("down", '{"foo": "unexpected"}'),
        ("down", 'notjson'),
    ])
    def test_http_json(self, expected_status, body):
        """
        Test JSON response checks
        """
        server = self.start_server(body, 200)
        try:
            self.render_config_template(
                monitors=[{
                    "type": "http",
                    "urls": ["http://localhost:{}".format(server.server_port)],
                    "check_response_json": [{
                        "description": "foo equals bar",
                        "condition": {
                            "equals": {"foo": {"baz": "bar"}}
                        }
                    }]
                }]
            )

            try:
                proc = self.start_beat()
                self.wait_until(lambda: self.log_contains("heartbeat is running"))

                self.wait_until(
                    lambda: self.output_has(lines=1))
            finally:
                proc.check_kill_and_wait()

            self.assert_last_status(expected_status)
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
                raise SkipTest
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
