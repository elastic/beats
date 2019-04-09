import os
import metricbeat
import unittest
from nose.plugins.attrib import attr
import requests
import time

HTTP_FIELDS = metricbeat.COMMON_FIELDS + ["http"]


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['http']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_json(self):
        """
        http json metricset test
        """
        self.render_config_template(modules=[{
            "name": "http",
            "metricsets": ["json"],
            "hosts": [self.get_host()],
            "period": "5s",
            "namespace": "test",
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        assert evt["http"]["test"]["hello"] == "world"

        del evt["http"]["test"]["hello"]

        self.assertItemsEqual(self.de_dot(HTTP_FIELDS), evt.keys(), evt)

        self.assert_fields_are_documented(evt)

    def test_server(self):
        """
        http server metricset test
        """
        port = 8082
        host = "localhost"
        self.render_config_template(modules=[{
            "name": "http",
            "metricsets": ["server"],
            "port": port,
            "host": host,
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("Starting HTTP"))
        requests.post("http://" + host + ":" + str(port),
                      json={'hello': 'world'}, headers={'Content-Type': 'application/json'})
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        assert evt["http"]["server"]["hello"] == "world"

        # Delete dynamic namespace part for fields comparison
        del evt["http"]["server"]

        self.assertItemsEqual(self.de_dot(HTTP_FIELDS), evt.keys(), evt)

        self.assert_fields_are_documented(evt)

    def get_host(self):
        return "http://" + os.getenv('HTTP_HOST', 'localhost') + ':' + os.getenv('HTTP_PORT', '8080')
