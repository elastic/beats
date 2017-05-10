import os
import sys

sys.path.append(os.path.join(os.path.dirname(__file__), '../../../libbeat/tests/system'))

from beat.beat import TestCase


class BaseTest(TestCase):
    @classmethod
    def setUpClass(self):
        self.beat_name = "heartbeat"
        super(BaseTest, self).setUpClass()
