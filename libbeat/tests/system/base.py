import os
from beat.beat import TestCase
from elasticsearch import Elasticsearch, NotFoundError


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
        self._es = None
        super(BaseTest, self).setUpClass()

    def esClient(self):
        if self._es:
            return self._es

        self._es = Elasticsearch([self.get_elasticsearch_url()])
        return self._es

    # stolen from filebeat/tests/system/filebeat.py
    @property
    def logs(self):
        return self.log_access()

    # stolen from filebeat/tests/system/filebeat.py
    def log_access(self, file=None):
        file = file if file else self.beat_name + ".log"
        return LogState(os.path.join(self.working_dir, file))
