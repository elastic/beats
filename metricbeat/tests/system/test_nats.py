import os
import metricbeat
import unittest

NATS_FIELDS = metricbeat.COMMON_FIELDS + ["nats"]


class TestNats(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['nats']

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
        self.assertEqual(len(output), 1)
        evt = output[0]

        self.assertItemsEqual(self.de_dot(NATS_FIELDS), evt.keys(), evt)

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
        self.assertEqual(len(output), 1)
        evt = output[0]

        self.assertItemsEqual(self.de_dot(NATS_FIELDS), evt.keys(), evt)

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
        self.assertEqual(len(output), 1)
        evt = output[0]

        self.assertItemsEqual(self.de_dot(NATS_FIELDS), evt.keys(), evt)

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
        self.assertEqual(len(output), 1)
        evt = output[0]

        self.assertItemsEqual(self.de_dot(NATS_FIELDS), evt.keys(), evt)

        self.assert_fields_are_documented(evt)


class TestNats1_3(TestNats):
    COMPOSE_SERVICES = ['nats_1_3']
