import os
import sys
import unittest

sys.path.append(os.path.join(os.path.dirname(__file__), '../../tests/system'))
from xpack_metricbeat import XPackTest, metricbeat


MSSQL_FIELDS = metricbeat.COMMON_FIELDS + ["mssql"]

MSSQL_STATUS_FIELDS = ["clients", "cluster", "cpu", "keyspace", "memory",
                       "persistence", "replication", "server", "stats"]


class Test(XPackTest):

    COMPOSE_SERVICES = ['mssql']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_status(self):
        """
        MSSQL module outputs an event.
        """
        self.render_config_template(modules=[{
            "name": "mssql",
            "metricsets": ["transaction_log"],
            "hosts": self.get_hosts(),
            "username": self.get_username(),
            "password": self.get_password(),
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 4)
        evt = output[0]

        self.assertCountEqual(self.de_dot(MSSQL_FIELDS), evt.keys())
        self.assertTrue(evt["mssql"]["transaction_log"]["space_usage"]["used"]["pct"] > 0)
        self.assertTrue(evt["mssql"]["transaction_log"]["stats"]["active_size"]["bytes"] > 0)

        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_performance(self):
        """
        MSSQL module outputs an event.
        """
        self.render_config_template(modules=[{
            "name": "mssql",
            "metricsets": ["performance"],
            "hosts": self.get_hosts(),
            "username": self.get_username(),
            "password": self.get_password(),
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        self.assertCountEqual(self.de_dot(MSSQL_FIELDS), evt.keys())
        self.assertTrue(evt["mssql"]["performance"]["buffer"]["page_life_expectancy"]["sec"] > 0)
        self.assertTrue(evt["mssql"]["performance"]["user_connections"] > 0)

        self.assert_fields_are_documented(evt)

    def get_username(self):
        return os.getenv('MSSQL_USERNAME', 'SA')

    def get_password(self):
        return os.getenv('MSSQL_PASSWORD', '1234_asdf')
