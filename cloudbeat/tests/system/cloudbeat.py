import os
import sys
from beat.beat import TestCase


class BaseTest(TestCase):

    @classmethod
    def setUpClass(self):
        self.beat_name = "cloudbeat"
        self.beat_path = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../"))
        super(BaseTest, self).setUpClass()
