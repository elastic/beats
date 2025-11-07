import metricbeat
import os
import sys
import unittest
from pkg_resources import packaging
from parameterized import parameterized

NATS_FIELDS = metricbeat.COMMON_FIELDS + ["nats"]


@metricbeat.parameterized_with_supported_versions
class TestNats(metricbeat.BaseTest):
    COMPOSE_SERVICES = ['nats', 'nats-routes']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_stats(self):
        """
        nats stats test
        """
        self.render_config_template(modules=[{
            "name": "nats",
            "metricsets": ["stats"],
            "hosts": self.get_hosts(),
            "period": "5s",
            "stats.metrics_path": "/varz"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 2)
        evt = output[0]

        self.assertCountEqual(self.de_dot(NATS_FIELDS), evt.keys(), evt)

        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_connections(self):
        """
        nats connections test
        """
        self.render_config_template(modules=[{
            "name": "nats",
            "metricsets": ["connections"],
            "hosts": self.get_hosts(),
            "period": "5s",
            "connections.metrics_path": "/connz"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 2)
        evt = output[0]

        self.assertCountEqual(self.de_dot(NATS_FIELDS), evt.keys(), evt)

        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_connection(self):
        """
        nats connection test
        """
        self.render_config_template(modules=[{
            "name": "nats",
            "metricsets": ["connection"],
            "hosts": self.get_hosts(),
            "period": "5s",
            "connections.metrics_path": "/connz"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 2)
        evt = output[0]

        self.assertCountEqual(self.de_dot(NATS_FIELDS), evt.keys(), evt)

        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_routes(self):
        """
        nats routes test
        """
        self.render_config_template(modules=[{
            "name": "nats",
            "metricsets": ["routes"],
            "hosts": self.get_hosts(),
            "period": "5s",
            "routes.metrics_path": "/routez"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 2)
        evt = output[0]

        self.assertCountEqual(self.de_dot(NATS_FIELDS), evt.keys(), evt)

        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_route(self):
        """
        nats route test
        """
        self.render_config_template(modules=[{
            "name": "nats",
            "metricsets": ["route"],
            "hosts": self.get_hosts(),
            "period": "5s",
            "routes.metrics_path": "/routez"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()

        nats_version = self.COMPOSE_ENV["NATS_VERSION"]

        # There is a difference in the number of route events reported from versions older
        # than 2.10.
        if packaging.version.parse(nats_version) < packaging.version.parse("2.10.0"):
            self.assertEqual(len(output), 2)
        else:
            self.assertEqual(len(output), 8)

        evt = output[0]

        self.assertCountEqual(self.de_dot(NATS_FIELDS), evt.keys(), evt)

        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_subscriptions(self):
        """
        nats subscriptions test
        """
        self.render_config_template(modules=[{
            "name": "nats",
            "metricsets": ["subscriptions"],
            "hosts": self.get_hosts(),
            "period": "5s",
            "subscriptions.metrics_path": "/subsz"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 2)
        evt = output[0]

        self.assertCountEqual(self.de_dot(NATS_FIELDS), evt.keys(), evt)

        self.assert_fields_are_documented(evt)

    @parameterized.expand([
        "stats",
        "account",
        "stream",
        "consumer"
    ])
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_jetstream(self, category):
        """
        nats jetstream test
        """
        # There were no consumer stats available prior to 2.9, so this test won't pass.
        nats_version = self.COMPOSE_ENV["NATS_VERSION"]
        if category == "consumer" and packaging.version.parse(nats_version) < packaging.version.parse("2.9.25"):
            return

        self.render_config_template(modules=[{
            "name": "nats",
            "metricsets": ["jetstream"],
            "hosts": self.get_hosts(),
            "period": "5s",
            "extras": {
                "jetstream": {
                    "stats": {
                        "enabled": category == "stats"
                    },
                    "account": {
                        "enabled": category == "account"
                    },
                    "stream": {
                        "enabled": category == "stream"
                    },
                    "consumer": {
                        "enabled": category == "consumer"
                    },
                },
            }
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=20)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        for evt in output:
            self.assertEqual(evt["nats"]["jetstream"]["category"], category)
            self.assertCountEqual(self.de_dot(NATS_FIELDS), evt.keys(), evt)
            self.assert_fields_are_documented(evt)

    def get_hosts(self):
        return [self.compose_host("nats"), self.compose_host("nats-routes")]
