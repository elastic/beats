import os
from beat.beat import TestCase


class BaseTest(TestCase):

    @classmethod
    def setUpClass(self):
        self.beat_name = "mockbeat"
        self.beat_path = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../"))
        self.test_binary = self.beat_path + "/libbeat.test"
        self.beats = [
            "filebeat",
            "heartbeat",
            "metricbeat",
            "packetbeat",
            "winlogbeat"
        ]
        super(BaseTest, self).setUpClass()
