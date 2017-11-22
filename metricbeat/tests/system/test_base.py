import re
import sys
import unittest
from metricbeat import BaseTest
import time


class Test(BaseTest):

    @unittest.skipUnless(re.match("(?i)win|linux|darwin|freebsd|openbsd", sys.platform), "os")
    def test_start_stop(self):
        """
        Metricbeat starts and stops without error.
        """
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["cpu"],
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("start running"))
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        # Ensure all Beater stages are used.
        assert self.log_contains("Setup Beat: metricbeat")
        assert self.log_contains("metricbeat start running")
        assert self.log_contains("metricbeat stopped")

    #COMPOSE_SERVICES = ['elasticsearch']
#
    #@unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_template(self):
        """
        Test that the template can be loaded with `setup --template`
        """
        self.render_config_template(
            elasticsearch={"hosts": "localhost:9200"},
        )
        proc = self.start_beat(extra_args=["setup", "--template"])

        time.sleep(5)

        # Wait until template is loaded
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertTrue(len(output) >= 1)
        evt = output[0]
        print(evt)

        self.assert_fields_are_documented(evt)
