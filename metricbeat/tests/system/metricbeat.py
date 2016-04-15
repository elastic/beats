import sys

sys.path.append('../../../libbeat/tests/system')
from beat.beat import TestCase

COMMON_FIELDS = ["@timestamp", "beat", "metricset", "metricset-host",
                 "module", "rtt", "type"]


class BaseTest(TestCase):
    @classmethod
    def setUpClass(self):
        self.beat_name = "metricbeat"
        self.build_path = "../../build/system-tests/"
        self.beat_path = "../../metricbeat.test"

    def assert_fields_are_documented(self, evt):
        """
        Assert that all keys present in evt are documented in fields.yml.
        """
        expected_fields, _ = self.load_fields()
        flat = self.flatten_object(evt, [])

        for key in flat.keys():
            if key not in expected_fields:
                raise Exception("Key '{}' found in event is not documented!".format(key))
