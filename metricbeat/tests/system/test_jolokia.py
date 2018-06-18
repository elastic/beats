import os
import metricbeat
import unittest
from nose.plugins.attrib import attr
from parameterized import parameterized


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['jolokia']

    @parameterized.expand([
        'java.lang:name=PS MarkSweep,type=GarbageCollector',
        'java.lang:type=GarbageCollector,name=PS MarkSweep',
        'java.lang:name=*,type=GarbageCollector',
        'java.lang:type=GarbageCollector,name=*',
    ])
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_jmx(self, mbean):
        """
        jolokia jmx  metricset test
        """

        additional_content = """
  jmx.mappings:
    - mbean: '%s'
      attributes:
         - attr: CollectionCount
           field: gc.collection_count
""" % (mbean)

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

        assert evt["jolokia"]["test"]["gc"]["collection_count"] >= 0

    def get_hosts(self):
        return [os.getenv('JOLOKIA_HOST', 'localhost') + ':' +
                os.getenv('JOLOKIA_PORT', '8778')]
