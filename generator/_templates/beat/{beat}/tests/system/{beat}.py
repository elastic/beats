import os
import sys

# if the beats folder is removed use the one from the module cache
# (output of go list -m -f '{{.Dir}}' github.com/elastic/beats/v7)
sys.path.append('{es_beats}/libbeat/tests/system')

from beat.beat import TestCase


class BaseTest(TestCase):

    @classmethod
    def setUpClass(self):
        self.beat_name = "{beat}"
        self.beat_path = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../"))
        super(BaseTest, self).setUpClass()
