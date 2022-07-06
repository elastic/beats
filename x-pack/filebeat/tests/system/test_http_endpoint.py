import jinja2
import requests
import platform
import sys
import hmac
import hashlib
import os
import json
import ast
from filebeat import BaseTest
from requests.auth import HTTPBasicAuth
import unittest


class Test(BaseTest):
    """
    Test filebeat with the http_endpoint input
    """
    @classmethod
    def setUpClass(self):
        self.beat_name = "filebeat"
        self.beat_path = os.path.abspath(
            os.path.join(os.path.dirname(__file__), "../../"))

        super(BaseTest, self).setUpClass()

    def setUp(self):
        super(BaseTest, self).setUp()

        # Hack to make jinja2 have the right paths
        self.template_env = jinja2.Environment(
            loader=jinja2.FileSystemLoader([
                os.path.abspath(os.path.join(
                    self.beat_path, "../../filebeat")),
                os.path.abspath(os.path.join(self.beat_path, "../../libbeat"))
            ])
        )

    def get_config(self, options=None):
        """
        General function so that we do not have to define settings each time
        """
        host = "127.0.0.1"
        port = 8081
        input_raw = """
- type: http_endpoint
  enabled: true
  listen_address: {}
  listen_port: {}
"""
        if options:
            input_raw = '\n'.join([input_raw, options])
        self.beat_name = "filebeat"
        self.beat_path = os.path.abspath(
            os.path.join(os.path.dirname(__file__), "../../"))

        input_raw = input_raw.format(host, port)
        self.render_config_template(
            input_raw=input_raw,
            inputs=False,
        )
        self.host = host
        self.port = port
        self.prefix = 'testmessage'
        self.url = "http://{}:{}/".format(host, port)

    def test_http_endpoint_request(self):
        """
        Test http_endpoint input with HTTP events.
        """
        self.get_config()
        filebeat = self.start_beat()
        self.wait_until(lambda: self.log_contains(
            "Starting HTTP server on {}:{}".format(self.host, self.port)))

        message = "somerandommessage"
        payload = {self.prefix: message}
        headers = {"Content-Type": "application/json",
                   "Accept": "application/json"}
        r = requests.post(self.url, headers=headers, data=json.dumps(payload))

        self.wait_until(lambda: self.output_count(lambda x: x >= 1))
        filebeat.check_kill_and_wait()

        output = self.read_output()

        print("response:", r.status_code, r.text)

        assert r.text == '{"message": "success"}'
        assert output[0]["input.type"] == "http_endpoint"
        assert output[0]["json.{}".format(self.prefix)] == message

    def test_http_endpoint_request_multiple_documents(self):
        """
        Test http_endpoint input with multiple documents on a single HTTP request.
        """
        self.get_config()
        filebeat = self.start_beat()
        self.wait_until(lambda: self.log_contains(
            "Starting HTTP server on {}:{}".format(self.host, self.port)))

        N = 10
        message = "somerandommessage_{}"
        payload = [{self.prefix: message.format(i)} for i in range(N)]
        headers = {"Content-Type": "application/json",
                   "Accept": "application/json"}
        r = requests.post(self.url, headers=headers, data=json.dumps(payload))

        self.wait_until(lambda: self.output_count(lambda x: x == N))
        filebeat.check_kill_and_wait()

        output = self.read_output()

        print("response:", r.status_code, r.text)

        assert r.text == '{"message": "success"}'

        assert len(output) == N
        for i in range(N):
            assert output[i]["input.type"] == "http_endpoint"
            assert output[i]["json.{}".format(
                self.prefix)] == message.format(i)

    def test_http_endpoint_request_ndjson(self):
        """
        Test http_endpoint input with multiple documents on a single HTTP request (NDJSON).
        """

        options = """
  content_type: application/x-ndjson
"""
        self.get_config(options)
        filebeat = self.start_beat()
        self.wait_until(lambda: self.log_contains(
            "Starting HTTP server on {}:{}".format(self.host, self.port)))

        N = 10
        message = "somerandommessage_{}"
        payload = "\n".join(
            [json.dumps({self.prefix: message.format(i)}) for i in range(N)])
        headers = {"Content-Type": "application/x-ndjson",
                   "Accept": "application/json"}
        r = requests.post(self.url, headers=headers, data=payload)

        self.wait_until(lambda: self.output_count(lambda x: x == N))
        filebeat.check_kill_and_wait()

        output = self.read_output()

        print("response:", r.status_code, r.text)

        assert r.text == '{"message": "success"}'

        assert len(output) == N
        for i in range(N):
            assert output[i]["input.type"] == "http_endpoint"
            assert output[i]["json.{}".format(
                self.prefix)] == message.format(i)

    def test_http_endpoint_wrong_content_header(self):
        """
        Test http_endpoint input with wrong content header.
        """
        self.get_config()
        filebeat = self.start_beat()
        self.wait_until(lambda: self.log_contains(
            "Starting HTTP server on {}:{}".format(self.host, self.port)))

        message = "somerandommessage"
        payload = {self.prefix: message}
        headers = {"Content-Type": "application/xml",
                   "Accept": "application/json"}
        r = requests.post(self.url, headers=headers, data=json.dumps(payload))

        filebeat.check_kill_and_wait()

        print("response:", r.status_code, r.text)

        assert r.status_code == 415
        assert r.json()[
            'message'] == 'wrong Content-Type header, expecting application/json'

    @unittest.skipIf(os.getenv("CI") is not None and platform.system() == 'Darwin', 'Flaky test: https://github.com/elastic/beats/issues/30028')
    def test_http_endpoint_missing_auth_value(self):
        """
        Test http_endpoint input with missing basic auth values.
        """
        options = """
  basic_auth: true
  username: testuser
  password:
"""
        self.get_config(options)
        filebeat = self.start_beat()
        self.wait_until(lambda: self.log_contains(
            "username and password required when basicauth is enabled"))
        filebeat.kill_and_wait()

    def test_http_endpoint_wrong_auth_value(self):
        """
        Test http_endpoint input with wrong basic auth values.
        """
        options = """
  basic_auth: true
  username: testuser
  password: testpassword
"""
        self.get_config(options)
        filebeat = self.start_beat()
        self.wait_until(lambda: self.log_contains(
            "Starting HTTP server on {}:{}".format(self.host, self.port)))

        message = "somerandommessage"
        payload = {self.prefix: message}
        headers = {"Content-Type": "application/json",
                   "Accept": "application/json"}
        r = requests.post(self.url, headers=headers, data=json.dumps(
            payload), auth=HTTPBasicAuth('testuser', 'qwerty'))

        filebeat.check_kill_and_wait()

        print("response:", r.status_code, r.text)

        assert r.status_code == 401
        assert r.json()['message'] == 'incorrect username or password'

    def test_http_endpoint_wrong_auth_header(self):
        """
        Test http_endpoint input with wrong auth header and secret.
        """
        options = """
  secret.header: Authorization
  secret.value: 123password
"""
        self.get_config(options)
        filebeat = self.start_beat()
        self.wait_until(lambda: self.log_contains(
            "Starting HTTP server on {}:{}".format(self.host, self.port)))

        message = "somerandommessage"
        payload = {self.prefix: message}
        headers = {"Content-Type": "application/json",
                   "Authorization": "password123"}
        r = requests.post(self.url, headers=headers, data=json.dumps(payload))

        filebeat.check_kill_and_wait()

        print("response:", r.status_code, r.text)

        assert r.status_code == 401
        assert r.json()['message'] == 'incorrect header or header secret'

    def test_http_endpoint_correct_auth_header(self):
        """
        Test http_endpoint input with correct auth header and secret.
        """
        options = """
  secret.header: Authorization
  secret.value: 123password
"""
        self.get_config(options)
        filebeat = self.start_beat()
        self.wait_until(lambda: self.log_contains(
            "Starting HTTP server on {}:{}".format(self.host, self.port)))

        message = "somerandommessage"
        payload = {self.prefix: message}
        headers = {"Content-Type": "application/json",
                   "Authorization": "123password"}
        r = requests.post(self.url, headers=headers, data=json.dumps(payload))

        filebeat.check_kill_and_wait()
        output = self.read_output()

        assert r.text == '{"message": "success"}'
        assert output[0]["input.type"] == "http_endpoint"
        assert output[0]["json.{}".format(self.prefix)] == message

    def test_http_endpoint_valid_hmac(self):
        """
        Test http_endpoint input with valid hmac signature.
        """
        options = """
  hmac.header: "X-Hub-Signature-256"
  hmac.key: "password123"
  hmac.type: "sha256"
  hmac.prefix: "sha256="
"""
        self.get_config(options)
        filebeat = self.start_beat()
        self.wait_until(lambda: self.log_contains(
            "Starting HTTP server on {}:{}".format(self.host, self.port)))

        message = "somerandommessage"
        payload = {self.prefix: message}

        h = hmac.new("password123".encode(), json.dumps(
            payload).encode(), hashlib.sha256)
        print(h.hexdigest())
        headers = {"Content-Type": "application/json",
                   "X-Hub-Signature-256": "sha256=" + h.hexdigest()}
        r = requests.post(self.url, headers=headers, data=json.dumps(payload))

        filebeat.check_kill_and_wait()
        output = self.read_output()

        assert r.text == '{"message": "success"}'
        assert output[0]["input.type"] == "http_endpoint"
        assert output[0]["json.{}".format(self.prefix)] == message

    def test_http_endpoint_invalid_hmac(self):
        """
        Test http_endpoint input with invalid hmac signature.
        """
        options = """
  hmac.header: "X-Hub-Signature-256"
  hmac.key: "password123"
  hmac.type: "sha256"
  hmac.prefix: "sha256="
"""
        self.get_config(options)
        filebeat = self.start_beat()
        self.wait_until(lambda: self.log_contains(
            "Starting HTTP server on {}:{}".format(self.host, self.port)))

        message = "somerandommessage"
        payload = {self.prefix: message}

        h = hmac.new("password321".encode(), json.dumps(
            payload).encode(), hashlib.sha256)
        headers = {"Content-Type": "application/json",
                   "X-Hub-Signature-256": "shad256=" + h.hexdigest()}
        r = requests.post(self.url, headers=headers, data=json.dumps(payload))

        filebeat.check_kill_and_wait()

        print("response:", r.status_code, r.text)

        assert r.status_code == 401
        self.assertRegex(r.json()['message'], 'invalid HMAC signature')

    def test_http_endpoint_preserve_original_event(self):
        """
        Test http_endpoint input while preserving the original event.
        """
        options = """
  preserve_original_event: true
"""
        self.get_config(options)
        filebeat = self.start_beat()
        self.wait_until(lambda: self.log_contains(
            "Starting HTTP server on {}:{}".format(self.host, self.port)))

        payload = {self.prefix: "somerandommessage"}
        bc = json.dumps(payload, separators=(',', ':')).encode('utf-8')
        headers = {"Content-Type": "application/json"}
        r = requests.post(self.url, headers=headers, data=json.dumps(payload))

        filebeat.check_kill_and_wait()
        output = self.read_output()

        assert r.status_code == 200
        assert output[0]["event.original"].encode("utf-8") == bc

    def test_http_endpoint_include_headers_single_value(self):
        """
        Test http_endpoint input while including headers.
        """
        options = """
  include_headers: ["TestHeader"]
"""
        self.get_config(options)
        filebeat = self.start_beat()
        self.wait_until(lambda: self.log_contains(
            "Starting HTTP server on {}:{}".format(self.host, self.port)))

        message = "somerandommessage"
        payload = {self.prefix: message}
        headers = {"Content-Type": "application/json",
                   "TestHeader": "TestHeaderValue"}
        r = requests.post(self.url, headers=headers, data=json.dumps(payload))

        filebeat.check_kill_and_wait()
        output = self.read_output()

        assert r.status_code == 200
        assert output[0]["headers.Testheader"] == ['TestHeaderValue']

    def test_http_endpoint_include_headers_empty_value(self):
        """
        Test http_endpoint input while including headers that has an emnpty value.
        """
        options = """
  include_headers: ["TestHeader"]
"""
        self.get_config(options)
        filebeat = self.start_beat()
        self.wait_until(lambda: self.log_contains(
            "Starting HTTP server on {}:{}".format(self.host, self.port)))

        message = "somerandommessage"
        payload = {self.prefix: message}
        headers = {"Content-Type": "application/json", "TestHeader": ""}
        r = requests.post(self.url, headers=headers, data=json.dumps(payload))

        filebeat.check_kill_and_wait()
        output = self.read_output()

        assert r.status_code == 200
        assert output[0]["headers.Testheader"] == [""]

    def test_http_endpoint_include_headers_without_header(self):
        """
        Test http_endpoint input while including headers, while the header is not in the request.
        """
        options = """
  include_headers: ["TestHeader"]
"""
        self.get_config(options)
        filebeat = self.start_beat()
        self.wait_until(lambda: self.log_contains(
            "Starting HTTP server on {}:{}".format(self.host, self.port)))

        message = "somerandommessage"
        payload = {self.prefix: message}
        headers = {"Content-Type": "application/json"}
        r = requests.post(self.url, headers=headers, data=json.dumps(payload))

        filebeat.check_kill_and_wait()
        output = self.read_output()

        assert r.status_code == 200
        assert not output[0].get("headers")

    def test_http_endpoint_include_headers_not_canonical_config(self):
        """
        Test http_endpoint input while including headers, while the header in config is not canonical.
        """
        options = """
  include_headers: ["test-header"]
"""
        self.get_config(options)
        filebeat = self.start_beat()
        self.wait_until(lambda: self.log_contains(
            "Starting HTTP server on {}:{}".format(self.host, self.port)))

        message = "somerandommessage"
        payload = {self.prefix: message}
        headers = {"Content-Type": "application/json",
                   "Test-Header": "TestHeaderValue"}
        r = requests.post(self.url, headers=headers, data=json.dumps(payload))

        filebeat.check_kill_and_wait()
        output = self.read_output()

        assert r.status_code == 200
        assert output[0]["headers.Test-Header"] == ["TestHeaderValue"]

    def test_http_endpoint_include_headers_not_canonical_header(self):
        """
        Test http_endpoint input while including headers, while the header in request is not canonical.
        """
        options = """
  include_headers: ["test-header"]
"""
        self.get_config(options)
        filebeat = self.start_beat()
        self.wait_until(lambda: self.log_contains(
            "Starting HTTP server on {}:{}".format(self.host, self.port)))

        message = "somerandommessage"
        payload = {self.prefix: message}
        headers = {"Content-Type": "application/json",
                   "test-header": "TestHeaderValue"}
        r = requests.post(self.url, headers=headers, data=json.dumps(payload))

        filebeat.check_kill_and_wait()
        output = self.read_output()

        assert r.status_code == 200
        assert output[0]["headers.Test-Header"] == ["TestHeaderValue"]

    def test_http_endpoint_empty_body(self):
        """
        Test http_endpoint input with empty body.
        """
        self.get_config()
        filebeat = self.start_beat()
        self.wait_until(lambda: self.log_contains(
            "Starting HTTP server on {}:{}".format(self.host, self.port)))

        headers = {"Content-Type": "application/json",
                   "Accept": "application/json"}
        r = requests.post(self.url, headers=headers, data="")

        filebeat.check_kill_and_wait()

        print("response:", r.status_code, r.text)

        assert r.status_code == 406
        assert r.json()['message'] == 'body cannot be empty'

    def test_http_endpoint_malformed_json(self):
        """
        Test http_endpoint input with malformed body.
        """

        self.get_config()
        filebeat = self.start_beat()
        self.wait_until(lambda: self.log_contains(
            "Starting HTTP server on {}:{}".format(self.host, self.port)))
        payload = '{"message::":: "something"}'
        headers = {"Content-Type": "application/json",
                   "Accept": "application/json"}
        r = requests.post(self.url, headers=headers, data=payload)

        filebeat.check_kill_and_wait()

        print("response:", r.status_code, r.text)

        assert r.status_code == 400
        self.assertRegex(r.json()['message'], 'malformed JSON')

    def test_http_endpoint_get_request(self):
        """
        Test http_endpoint input with GET request.
        """

        self.get_config()
        filebeat = self.start_beat()
        self.wait_until(lambda: self.log_contains(
            "Starting HTTP server on {}:{}".format(self.host, self.port)))
        message = "somerandommessage"
        payload = {self.prefix: message}
        headers = {"Content-Type": "application/json",
                   "Accept": "application/json"}
        r = requests.get(self.url, headers=headers, data=json.dumps(payload))
        filebeat.check_kill_and_wait()

        print("response:", r.status_code, r.text)

        assert r.status_code == 405
        assert r.json()['message'] == 'only POST requests are allowed'
