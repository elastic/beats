import os
from datetime import datetime
from beat.beat import TestCase
from elasticsearch import Elasticsearch, NotFoundError


class BaseTest(TestCase):
    today = datetime.now().strftime("%Y%m%d")

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
        self._es = None
        super(BaseTest, self).setUpClass()

    def es_client(self):
        if self._es:
            return self._es

        self._es = Elasticsearch([self.get_elasticsearch_url()])
        return self._es
