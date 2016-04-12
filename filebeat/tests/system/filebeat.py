import json
import os
import sys

sys.path.append('../../../libbeat/tests/system')

from beat.beat import TestCase


class BaseTest(TestCase):

    @classmethod
    def setUpClass(self):
        self.beat_name = "filebeat"
        super(BaseTest, self).setUpClass()

    def get_registry(self):
        # Returns content of the registry file
        dotFilebeat = self.working_dir + '/registry'
        assert os.path.isfile(dotFilebeat) is True

        with open(dotFilebeat) as file:
            return json.load(file)
