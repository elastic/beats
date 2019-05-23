import os
import metricbeat
import unittest
from nose.plugins.attrib import attr

HAPROXY_FIELDS = metricbeat.COMMON_FIELDS + ["haproxy"]


class HaproxyTest(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['haproxy']

    def _test_info(self):
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        self.assertItemsEqual(self.de_dot(HAPROXY_FIELDS + ["process"]), evt.keys(), evt)

        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_info_socket(self):
        """
        haproxy info unix socket metricset test
        """
        self.render_config_template(modules=[{
            "name": "haproxy",
            "metricsets": ["info"],
            "hosts": ["tcp://%s" % (self.compose_host(port="14567/tcp"))],
            "period": "5s"
        }])
        self._test_info()

    def _test_stat(self):
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            print(evt)
            self.assertItemsEqual(self.de_dot(HAPROXY_FIELDS + ["process"]), evt.keys(), evt)
            self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_stat_socket(self):
        """
        haproxy stat unix socket metricset test
        """
        self.render_config_template(modules=[{
            "name": "haproxy",
            "metricsets": ["stat"],
            "hosts": ["tcp://%s" % (self.compose_host(port="14567/tcp"))],
            "period": "5s"
        }])
        self._test_stat()

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_stat_http(self):
        """
        haproxy stat http metricset test
        """
        self.render_config_template(modules=[{
            "name": "haproxy",
            "metricsets": ["stat"],
            "hosts": ["http://%s/stats" % (self.compose_host(port="14568/tcp"))],
            "period": "5s"
        }])
        self._test_stat()

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_stat_http_auth(self):
        """
        haproxy stat http basic auth metricset test
        """
        self.render_config_template(modules=[{
            "name": "haproxy",
            "metricsets": ["stat"],
            "username": "admin",
            "password": "admin",
            "hosts": ["http://%s/stats" % (self.compose_host(port="14569/tcp"))],
            "period": "5s"
        }])
        self._test_stat()


class Haproxy_1_6_Test(HaproxyTest):
    COMPOSE_SERVICES = ['haproxy_1_6']


class Haproxy_1_7_Test(HaproxyTest):
    COMPOSE_SERVICES = ['haproxy_1_7']
