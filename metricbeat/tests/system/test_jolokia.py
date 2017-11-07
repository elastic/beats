import os
import metricbeat
import unittest
from nose.plugins.attrib import attr


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['jolokia']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_jmx(self):
        """
        jolokia jmx  metricset test
        """

        additional_content = """
  jmx.mappings:
    - mbean: 'java.lang:type=Runtime'
      attributes:
         - attr: Uptime
           field: uptime
"""

        self.render_config_template(modules=[{
            "name": "jolokia",
            "metricsets": ["jmx"],
            "hosts": self.get_hosts(),
            "period": "1s",
            "namespace": "test",
            "additional_content": additional_content,
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=20)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertTrue(len(output) >= 1)
        evt = output[0]
        print(evt)

        assert evt["jolokia"]["test"]["uptime"] > 0

    def get_hosts(self):
        return [os.getenv('JOLOKIA_HOST', 'localhost') + ':' +
                os.getenv('JOLOKIA_PORT', '8778')]
