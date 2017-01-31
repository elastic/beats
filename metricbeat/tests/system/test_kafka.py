import os
import metricbeat
import unittest
from nose.plugins.attrib import attr


class Test(metricbeat.BaseTest):

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_partition(self):
        """
        kafka partition metricset test
        """
        self.render_config_template(modules=[{
            "name": "kafka",
            "metricsets": ["partition"],
            "hosts": self.get_hosts(),
            "period": "1s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=20)
        proc.check_kill_and_wait()

        output = self.read_output_json()
        self.assertTrue(len(output) >= 1)
        evt = output[0]
        print(evt)

        self.assert_fields_are_documented(evt)

    def get_hosts(self):
        return [os.getenv('KAFKA_HOST', 'localhost') + ':' +
                os.getenv('KAFKA_PORT', '9092')]
