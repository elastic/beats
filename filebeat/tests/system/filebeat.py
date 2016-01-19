import json
import os
import sys

sys.path.append('../../../libbeat/tests/system')

from beat.beat import TestCase


class BaseTest(TestCase):

    @classmethod
    def setUpClass(self):
        self.beat_name = "filebeat"
        self.build_path = "../../build/system-tests/"
        self.beat_path = "../../filebeat.test"

    def get_dot_filebeat(self):
        # Returns content of the .filebeat file
        dotFilebeat = self.working_dir + '/.filebeat'
        assert os.path.isfile(dotFilebeat) is True

        with open(dotFilebeat) as file:
            return json.load(file)
