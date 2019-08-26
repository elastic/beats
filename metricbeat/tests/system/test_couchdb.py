import os
import metricbeat
import unittest


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['couchdb']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_stats(self):
        """
        Couchdb module outputs an event.
        """
        self.render_config_template(modules=[{
            "name": "couchdb",
            "metricsets": ["server"],
            "hosts": self.get_hosts(),
            "period": "5s",
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=20)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertTrue(len(output) >= 1)
        event = output[0]
        print(event)

        self.assertNotIn("error", event)

        self.assert_fields_are_documented(event)
