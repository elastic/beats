import os
import sys
import unittest

sys.path.append(os.path.join(os.path.dirname(__file__), '../../tests/system'))
from xpack_metricbeat import XPackTest, metricbeat


class Test(XPackTest):

    COMPOSE_SERVICES = ['ibmmq']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_qmgr(self):
        """
        ibmmq qmgr test
        """
        self.render_config_template(modules=[{
            "name": "ibmmq",
            "metricsets": ["qmgr"],
            "hosts": self.get_hosts(),
            "period": "5s",
        }])
        proc = self.start_beat(home=self.beat_path)
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            self.assert_fields_are_documented(evt)
            self.assertIn("prometheus", evt.keys(), evt)
            self.assertIn("metrics", evt["prometheus"].keys(), evt)
