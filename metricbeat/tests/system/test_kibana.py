import os
import metricbeat
import unittest
from nose.plugins.skip import SkipTest


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['elasticsearch', 'kibana']

    COMPOSE_TIMEOUT = 600

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_status(self):
        """
        kibana status metricset test
        """
        env = os.environ.get('TESTING_ENVIRONMENT')

        if env == "2x" or env == "5x":
            # Skip for 5.x and 2.x tests as Kibana endpoint not available
            raise SkipTest

        self.render_config_template(modules=[{
            "name": "kibana",
            "metricsets": ["status"],
            "hosts": self.get_hosts(),
            "period": "1s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=20)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertTrue(len(output) >= 1)
        evt = output[0]
        print evt

        self.assert_fields_are_documented(evt)

    def get_hosts(self):
        return [os.getenv('KIBANA_HOST', 'localhost') + ':' +
                os.getenv('KIBANA_PORT', '5601')]
