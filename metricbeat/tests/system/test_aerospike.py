import os
import metricbeat
import unittest
from nose.plugins.attrib import attr

AEROSPIKE_FIELDS = metricbeat.COMMON_FIELDS + ["aerospike"]


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['aerospike']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_namespace(self):
        """
        aerospike namespace metricset test
        """
        self.render_config_template(modules=[{
            "name": "aerospike",
            "metricsets": ["namespace"],
            "hosts": self.get_hosts(),
            "period": "1s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        self.assertItemsEqual(self.de_dot(AEROSPIKE_FIELDS), evt.keys())

        self.assert_fields_are_documented(evt)

    def get_hosts(self):
        return [os.getenv('AEROSPIKE_HOST', 'localhost') + ':' +
                os.getenv('AEROSPIKE_PORT', '3000')]
