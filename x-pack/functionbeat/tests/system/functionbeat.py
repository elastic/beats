import os
import sys
import base64

sys.path.append(os.path.join(os.path.dirname(__file__), '../../../../libbeat/tests/system'))
from beat.beat import TestCase


class BaseTest(TestCase):

    @classmethod
    def setUpClass(self):
        self.beat_name = "functionbeat"
        self.beat_path = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../"))
        super(BaseTest, self).setUpClass()

    def cloud_id(self):
        return os.environ.get("CLOUD_ID")

    def cloud_auth(self):
        return os.environ.get("CLOUD_AUTH")

    def cloud_to_elasticsearch_py(self):
        encoded = self.cloud_id().split(":")[1]
        decoded = base64.standard_b64decode(encoded)

        parts = decoded.split("$")
        if len(parts) < 3:
            raise "invalid cloud_id: %s" % decoded

        return "http://%s@%s.%s" % (self.cloud_auth(), parts[1], parts[0])

    def collect_messages(self, results):
        messages = []
        for hit in results['hits']['hits']:
            messages.append(hit["_source"])

        return sorted(messages, key=lambda k: k['message'])
