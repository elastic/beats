import metricbeat
import os
import sys
import unittest


PROMETHEUS_FIELDS = metricbeat.COMMON_FIELDS + ["prometheus"]


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['prometheus']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_stats(self):
        """
        prometheus stats test
        """
        self.render_config_template(modules=[{
            "name": "prometheus",
            "metricsets": ["collector"],
            "hosts": self.get_hosts(),
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        evt = output[0]

        self.assertCountEqual(self.de_dot(PROMETHEUS_FIELDS), evt.keys(), evt)

        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_query(self):
        """
        prometheus query test
        """
        self.render_config_template(modules=[{
            "name": "prometheus",
            "metricsets": ["query"],
            "hosts": self.get_hosts(),
            "period": "5s",
            "extras": {
                "queries": [{
                    "path": "/api/v1/query",
                    'name': 'go_info',
                    'params': {'query': 'go_info'}
                }]
            }
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, 60)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        evt = output[0]

        self.assertCountEqual(self.de_dot(PROMETHEUS_FIELDS), evt.keys(), evt)

        self.assert_fields_are_documented(evt)


class TestRemoteWrite(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['prometheus-host-network']

    @unittest.skip("use of host network incompatible with docker update: https://github.com/elastic/beats/issues/38854")
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_remote_write(self):
        """
        prometheus remote_write test
        """
        self.render_config_template(modules=[{
            "name": "prometheus",
            "metricsets": ["remote_write"],
            "period": "5s",
            "host": "localhost",
            "port": "9201"
        }])
        proc = self.start_beat()

        self.wait_until(lambda: self.log_contains("Starting HTTP"))

        self.wait_until(lambda: self.output_lines() > 0, 60)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        evt = output[0]

        self.assertCountEqual(self.de_dot(PROMETHEUS_FIELDS), evt.keys(), evt)
        self.assert_fields_are_documented(evt)
