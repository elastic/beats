from beat.beat import TestCase


class BaseTest(TestCase):

    @classmethod
    def setUpClass(self):
        self.beat_name = "mockbeat"
        self.build_path = "../../build/system-tests/"
        self.beat_path = "../../libbeat.test"
