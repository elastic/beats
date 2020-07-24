import json
import jinja2
import os
import random
import sys

from datetime import datetime
from flask import Flask, jsonify, request
from multiprocessing import Process

sys.path.append(os.path.join(os.path.dirname(__file__),
                             '../../../../filebeat/tests/system'))

from filebeat import BaseTest


class Test(BaseTest):
    """
    Test filebeat with the httpjson input
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

    def set_config(self, extra_options=[]):
        """
        General function so that we do not have to define settings each time
        """
        options = ["- type: httpjson", "enabled: true"]
        options.extend(extra_options)

        self.render_config_template(
            input_raw='\n  '.join(options),
            inputs=False,
        )

    def start_server(self, method, handler, ssl=False):
        """
        Creates a new http test server that will respond with the given handler
        """
        app = Flask(__name__)
        app.app_context().push()

        app.route('/', methods=[method])(handler)

        kwargs = {}
        if ssl:
            kwargs = {"ssl_context": "adhoc"}

        process = Process(target=app.run, kwargs=kwargs)

        def shutdown():
            app.do_teardown_appcontext()
            process.terminate()
            process.join()

        process.start()

        return shutdown

    def test_get(self):
        """
        Test httpjson input performs a simple GET request correctly.
        """

        message = {"hello": "world"}

        def handler():
            return jsonify(message)

        shutdown_func = self.start_server("GET", handler)

        options = [
            "http_method: GET",
            "interval: 0",
            "url: http://localhost:5000"
        ]
        self.set_config(options)

        filebeat = self.start_beat()

        self.wait_until(lambda: self.output_count(lambda x: x == 1))

        filebeat.check_kill_and_wait()
        shutdown_func()

        output = self.read_output()

        assert output[0]["input.type"] == "httpjson"
        assert json.loads(output[0]["message"]) == message

    def test_get_https(self):
        """
        Test httpjson input performs a simple GET request with HTTPS correctly.
        """

        message = {"hello": "world"}

        def handler():
            return jsonify(message)

        shutdown_func = self.start_server("GET", handler, ssl=True)

        options = [
            "http_method: GET",
            "interval: 0",
            "url: https://localhost:5000",
            "ssl.verification_mode: none"
        ]
        self.set_config(options)

        filebeat = self.start_beat()

        self.wait_until(lambda: self.output_count(lambda x: x == 1))

        filebeat.check_kill_and_wait()
        shutdown_func()

        output = self.read_output()

        assert output[0]["input.type"] == "httpjson"
        assert json.loads(output[0]["message"]) == message

    def test_rate_limit_retry(self):
        """
        Test httpjson input performs a retry when is rate limited.
        """

        message = {"hello": "world"}

        is_retry = False

        def handler():
            nonlocal is_retry

            resp = jsonify(message)
            if is_retry:
                return resp

            is_retry = True
            resp.headers["X-Rate-Limit-Limit"] = "0"
            resp.headers["X-Rate-Limit-Remaining"] = "0"
            resp.headers["X-Rate-Limit-Reset"] = datetime.timestamp(
                datetime.now())
            resp.status_code = 429

            return resp

        shutdown_func = self.start_server("GET", handler)

        options = [
            "http_method: GET",
            "interval: 0",
            "url: http://localhost:5000"
        ]
        self.set_config(options)

        filebeat = self.start_beat()

        self.wait_until(lambda: self.output_count(lambda x: x == 1))

        filebeat.check_kill_and_wait()
        shutdown_func()

        output = self.read_output()

        assert output[0]["input.type"] == "httpjson"
        assert json.loads(output[0]["message"]) == message

    def test_error_retry(self):
        """
        Test httpjson input performs a retry when the request fails.
        """

        message = {"hello": "world"}

        retry_count = 0

        def handler():
            nonlocal retry_count

            resp = jsonify(message)
            if retry_count == 2:
                return resp

            retry_count += 1
            resp.status_code = random.randrange(500, 599)

            return resp

        shutdown_func = self.start_server("GET", handler)

        options = [
            "http_method: GET",
            "interval: 0",
            "url: http://localhost:5000"
        ]
        self.set_config(options)

        filebeat = self.start_beat()

        self.wait_until(lambda: self.output_count(lambda x: x == 1))

        filebeat.check_kill_and_wait()
        shutdown_func()

        output = self.read_output()

        assert output[0]["input.type"] == "httpjson"
        assert json.loads(output[0]["message"]) == message

    def test_array_response(self):
        """
        Test httpjson input parses properly an array response.
        """

        message = {
            "hello": [
                {
                    "foo": "bar",
                    "list": [
                        {"foo": "bar"},
                        {"hello": "world"}
                    ]
                },
                {
                    "foo": "bar",
                    "list": [
                        {"foo": "bar"}
                    ]
                }
            ]
        }

        def handler():
            return jsonify(message)

        shutdown_func = self.start_server("GET", handler)

        options = [
            "http_method: GET",
            "interval: 0",
            "url: http://localhost:5000",
            "json_objects_array: hello"
        ]
        self.set_config(options)

        filebeat = self.start_beat()

        self.wait_until(lambda: self.output_count(lambda x: x == 2))

        filebeat.check_kill_and_wait()
        shutdown_func()

        output = self.read_output()

        assert output[0]["input.type"] == "httpjson"
        assert json.loads(output[0]["message"]) == message["hello"][0]
        assert json.loads(output[1]["message"]) == message["hello"][1]

    def test_post(self):
        """
        Test httpjson input performs a simple POST request correctly.
        """

        message = {"hello": "world"}

        def handler():
            if request.get_json() != {"test": "abc"}:
                resp = jsonify({"error": "got {}".format(request.get_data())})
                resp.status_code = 400
                return resp
            return jsonify(message)

        shutdown_func = self.start_server("POST", handler)

        options = [
            "http_method: POST",
            "interval: 0",
            "url: http://localhost:5000",
            "http_request_body:",
            "  test: abc"
        ]
        self.set_config(options)

        filebeat = self.start_beat()

        self.wait_until(lambda: self.output_count(lambda x: x == 1))

        filebeat.check_kill_and_wait()
        shutdown_func()

        output = self.read_output()

        assert output[0]["input.type"] == "httpjson"
        assert json.loads(output[0]["message"]) == message

    def test_repeated_post(self):
        """
        Test httpjson input performs several POST requests correctly.
        """

        message = {"hello": "world"}

        def handler():
            if request.get_json() != {"test": "abc"}:
                resp = jsonify({"error": "got {}".format(request.get_data())})
                resp.status_code = 400
                return resp
            return jsonify(message)

        shutdown_func = self.start_server("POST", handler)

        options = [
            "http_method: POST",
            "interval: 300ms",
            "url: http://localhost:5000",
            "http_request_body:",
            "  test: abc"
        ]
        self.set_config(options)

        filebeat = self.start_beat()

        self.wait_until(lambda: self.output_count(lambda x: x == 3))

        filebeat.check_kill_and_wait()
        shutdown_func()

        output = self.read_output()

        assert output[0]["input.type"] == "httpjson"
        assert json.loads(output[0]["message"]) == message
        assert json.loads(output[1]["message"]) == message
        assert json.loads(output[2]["message"]) == message

    def test_oauth2(self):
        """
        Test httpjson input performs oauth2 requests correctly.
        """

        message = {"hello": "world"}
        is_oauth2_token_request = True

        def handler():
            nonlocal is_oauth2_token_request
            if is_oauth2_token_request:
                if request.method != "POST":
                    resp = jsonify({"error": "expected POST request"})
                    resp.status_code = 400
                    return resp
                if request.values["grant_type"] != "client_credentials":
                    resp = jsonify({"error": "expected grant_type was client_credentials"})
                    resp.status_code = 400
                    return resp
                if request.values["client_id"] != "a_client_id" or request.values["client_secret"] != "a_client_secret":
                    resp = jsonify({"error": "expected client credentials a_client_id:a_client_secret"})
                    resp.status_code = 400
                    return resp
                if request.values["scope"] != "scope1 scope2":
                    resp = jsonify({"error": "expected scope was scope1+scope2"})
                    resp.status_code = 400
                    return resp
                if request.values["param1"] != "v1":
                    resp = jsonify({"error": "expected param1 was v1"})
                    resp.status_code = 400
                    return resp
                is_oauth2_token_request = False
                return jsonify({"token_type": "Bearer", "expires_in": "60", "access_token": "abcd"})
            return jsonify(message)

        shutdown_func = self.start_server("POST", handler)

        options = [
            "http_method: POST",
            "interval: 0",
            "url: http://localhost:5000",
            "oauth2.client.id: a_client_id",
            "oauth2.client.secret: a_client_secret",
            "oauth2.token_url: http://localhost:5000",
            "oauth2.endpoint_params:",
            "  param1: v1",
            "oauth2.scopes: [scope1, scope2]"
        ]
        self.set_config(options)

        filebeat = self.start_beat()

        self.wait_until(lambda: self.output_count(lambda x: x == 1))

        filebeat.check_kill_and_wait()
        shutdown_func()

        output = self.read_output()

        assert output[0]["input.type"] == "httpjson"
        assert json.loads(output[0]["message"]) == message

    def test_split_events_by(self):
        """
        Test httpjson input splits events by key correctly.
        """

        message = {
            "hello": "world",
            "embedded": {
                "hello": "world",
            },
            "list": [
                {"foo": "bar"},
                {"hello": "world"}
            ]
        }

        def handler():
            return jsonify(message)

        shutdown_func = self.start_server("GET", handler)

        options = [
            "http_method: GET",
            "interval: 0",
            "url: http://localhost:5000",
            "split_events_by: list"
        ]
        self.set_config(options)

        filebeat = self.start_beat()

        self.wait_until(lambda: self.output_count(lambda x: x == 2))

        filebeat.check_kill_and_wait()
        shutdown_func()

        output = self.read_output()

        expected1 = {
            "hello": "world",
            "embedded": {
                "hello": "world",
            },
            "list": {"foo": "bar"}
        }

        expected2 = {
            "hello": "world",
            "embedded": {
                "hello": "world",
            },
            "list": {"hello": "world"}
        }
        assert output[0]["input.type"] == "httpjson"
        assert json.loads(output[0]["message"]) == expected1
        assert json.loads(output[1]["message"]) == expected2

    def test_split_events_by_not_found(self):
        """
        Test httpjson input does not fail when split key is not found
        """

        message = {"hello": "world"}

        def handler():
            return jsonify(message)

        shutdown_func = self.start_server("GET", handler)

        options = [
            "http_method: GET",
            "interval: 0",
            "url: http://localhost:5000",
            "split_events_by: list"
        ]
        self.set_config(options)

        filebeat = self.start_beat()

        self.wait_until(lambda: self.output_count(lambda x: x == 1))

        filebeat.check_kill_and_wait()
        shutdown_func()

        output = self.read_output()

        assert output[0]["input.type"] == "httpjson"
        assert json.loads(output[0]["message"]) == message

    def test_split_events_by_with_array(self):
        """
        Test httpjson input generate events when splitting from a key inside a list
        """

        message = {
            "objs": [
                {
                    "foo": "bar",
                    "list": [
                        {"bar": "baz"},
                        {"one": "two"}
                    ]
                },
                {"foo": "bar"}
            ]
        }

        def handler():
            return jsonify(message)

        shutdown_func = self.start_server("GET", handler)

        options = [
            "http_method: GET",
            "interval: 0",
            "url: http://localhost:5000",
            "json_objects_array: objs",
            "split_events_by: list"
        ]
        self.set_config(options)

        filebeat = self.start_beat()

        self.wait_until(lambda: self.output_count(lambda x: x == 3))

        filebeat.check_kill_and_wait()
        shutdown_func()

        output = self.read_output()

        expected1 = {
            "foo": "bar",
            "list": {"bar": "baz"}
        }

        expected2 = {
            "foo": "bar",
            "list": {"one": "two"}
        }

        expected3 = {"foo": "bar"}

        assert output[0]["input.type"] == "httpjson"
        assert json.loads(output[0]["message"]) == expected1
        assert json.loads(output[1]["message"]) == expected2
        assert json.loads(output[2]["message"]) == expected3

    def test_cursor(self):
        """
        Test httpjson input works correctly with a date cursor
        """

        message = [
            {"@timestamp": "2002-10-02T15:00:00Z", "foo": "bar"},
            {"@timestamp": "2002-10-02T15:00:01Z", "foo": "bar"}
        ]

        times = 0

        def handler():
            nonlocal times
            if times == 1:
                if request.values["$filter"] != "alertCreationTime ge 2002-10-02T15:00:01Z":
                    resp = jsonify({"error": "wrong filter"})
                    resp.status_code = 400
                    return resp
            times += 1
            return jsonify(message)

        shutdown_func = self.start_server("GET", handler)

        options = [
            "http_method: GET",
            "interval: 300ms",
            "url: http://localhost:5000",
            "date_cursor.field: \"@timestamp\"",
            "date_cursor.url_field: $filter",
            "date_cursor.value_template: alertCreationTime ge {{.}}",
            "date_cursor.initial_interval: 10m",
            "date_cursor.date_format: 2006-01-02T15:04:05Z",
        ]
        self.set_config(options)

        filebeat = self.start_beat()

        self.wait_until(lambda: self.output_count(lambda x: x == 4))

        filebeat.check_kill_and_wait()
        shutdown_func()

        output = self.read_output()

        assert output[0]["input.type"] == "httpjson"
        assert json.loads(output[0]["message"]) == message[0]
        assert json.loads(output[1]["message"]) == message[1]
        assert json.loads(output[2]["message"]) == message[0]
        assert json.loads(output[3]["message"]) == message[1]
