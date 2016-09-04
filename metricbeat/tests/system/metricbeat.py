import sys
import os

sys.path.append('../../../libbeat/tests/system')
from beat.beat import TestCase

COMMON_FIELDS = ["@timestamp", "beat", "metricset.name", "metricset.host",
                 "metricset.module", "metricset.rtt", "type"]

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)

class BaseTest(TestCase):
    @classmethod
    def setUpClass(self):
        self.beat_name = "metricbeat"
        self.build_path = "../../build/system-tests/"
        self.beat_path = "../../metricbeat.test"

    def assert_fields_are_documented(self, evt):
        """
        Assert that all keys present in evt are documented in fields.yml.
        This reads from the global fields.yml, means `make collect` has to be run before the check.
        """
        expected_fields, _ = self.load_fields()
        flat = self.flatten_object(evt, [])

        for key in flat.keys():
            if key not in expected_fields:
                raise Exception("Key '{}' found in event is not documented!".format(key))

    def de_dot(self, existing_fields):
        fields = {}

        # Dedot first level of dots
        for key in existing_fields:
            parts = key.split('.', 1)

            if len(parts) > 1:
                if parts[0] not in fields:
                    fields[parts[0]] = {}

                fields[parts[0]][parts[1]] = parts[1]
            else:
                fields[parts[0]] = parts[0]

        # Dedot further levels recursively
        for key in fields:
            if type(fields[key]) is dict:
                fields[key] = self.de_dot(fields[key])

        return fields
