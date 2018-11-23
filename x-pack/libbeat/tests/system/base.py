import sys
import os


sys.path.append(os.path.join(os.path.dirname(__file__),
                             '../../../../libbeat/tests/system'))


from beat.beat import TestCase


class BaseTest(TestCase):

    @classmethod
    def setUpClass(self):
        self.beat_name = "mockbeat"
        self.beat_path = os.path.abspath(
            os.path.join(os.path.dirname(__file__), "../../"))
        self.test_binary = self.beat_path + "/libbeat.test"
        super(BaseTest, self).setUpClass()
