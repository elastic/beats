import os
import sys

sys.path.append('../../../metricbeat/tests/system')
from metricbeat import BaseTest as MetricbeatTest


class BaseTest(MetricbeatTest):
    @classmethod
    def setUpClass(self):
        self.beat_name = "auditbeat"
        self.beat_path = os.path.abspath(
            os.path.join(os.path.dirname(__file__), "../../"))
        super(MetricbeatTest, self).setUpClass()
