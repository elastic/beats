import os
import sys
import unittest

sys.path.append(os.path.join(os.path.dirname(__file__), '../../tests/system'))
from xpack_metricbeat import XPackTest, metricbeat


class Test(XPackTest):

    COMPOSE_SERVICES = ['oracle']

    def get_hosts(self):
        return [self.compose_host(port='1521/tcp')+'/ORCLPDB1.localdomain']

    @metricbeat.tag('oracle')
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_performance(self):
        """
        oracle performance test
        """
        self.render_config_template(modules=[{
            "name": "oracle",
            "metricsets": ["performance"],
            "hosts": self.get_hosts(),
            "period": "5s",
            "username":"sys",
            "password":"Oradoc_db1"
        }])
        proc = self.start_beat(home=self.beat_path)
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            self.assert_fields_are_documented(evt)

    @metricbeat.tag('oracle')
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_tablespace(self):
        """
        oracle tablespace test
        """
        self.render_config_template(modules=[{
            "name": "oracle",
            "metricsets": ["tablespace"],
            "hosts": self.get_hosts(),
            "period": "5s",
            "username":"sys",
            "password":"Oradoc_db1"
        }])
        proc = self.start_beat(home=self.beat_path)
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            self.assert_fields_are_documented(evt)
