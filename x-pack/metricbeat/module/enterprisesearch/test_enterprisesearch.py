"""Integration tests for the Enterprise Search Metricbeat module"""

import os
import unittest
from xpack_metricbeat import XPackTest, metricbeat


class Test(XPackTest):
    COMPOSE_SERVICES = ['enterprise_search']
    COMPOSE_TIMEOUT = 600

    # -------------------------------------------------------------------------
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, 'integration test')
    def test_health(self):
        """Tests the Health API and the associated metricset"""

        # Setup the environment
        self.setup_environment(metricset="health")

        # Get a single event for testing
        evt = self.get_event()

        self.assertIn("enterprisesearch", evt)
        self.assertIn("health", evt["enterprisesearch"])

        health = evt["enterprisesearch"]["health"]
        self.assertIn("jvm", health)

    # -------------------------------------------------------------------------
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, 'integration test')
    def test_stats(self):
        """Tests the Stats API and the associated metricset"""

        # Setup the environment
        self.setup_environment(metricset="stats")

        # Get a single event for testing
        evt = self.get_event()

        self.assertIn("enterprisesearch", evt)
        self.assertIn("stats", evt["enterprisesearch"])

        stats = evt["enterprisesearch"]["stats"]
        self.assertIn("http", stats)

    # -------------------------------------------------------------------------
    def setup_environment(self, metricset):
        """Sets up the testing environment and starts all components needed"""

        self.render_config_template(modules=[{
            "name": "enterprisesearch",
            "metricsets": [metricset],
            "hosts": [self.compose_host(service="enterprise_search")],
            "username": self.get_username(),
            "password": self.get_password(),
            "period": "5s"
        }])

        proc = self.start_beat(home=self.beat_path)
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

    def get_event(self):
        """Gets a single event and checks that all fields are documented.
           Returns the event hash."""

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        self.assert_fields_are_documented(output[0])
        return output[0]

    @staticmethod
    def get_username():
        """Returns the user name to be used for Enterprise Search"""
        return os.getenv('ENT_SEARCH_USER', 'elastic')

    @staticmethod
    def get_password():
        """Returns the password to be used for Enterprise Search"""
        return os.getenv('ENT_SEARCH_PASSWORD', 'changeme')
