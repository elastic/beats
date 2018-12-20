import os
import sys

sys.path.append(os.path.join(os.path.dirname(__file__), '../../../../metricbeat/tests/system'))

import metricbeat


class XPackTest(metricbeat.BaseTest):

    @classmethod
    def setUpClass(self):
        self.beat_name = "metricbeat"
        self.beat_path = os.path.abspath(
            os.path.join(os.path.dirname(__file__), "../../"))

        super(XPackTest, self).setUpClass()

    def setUp(self):
        super(XPackTest, self).setUp()
