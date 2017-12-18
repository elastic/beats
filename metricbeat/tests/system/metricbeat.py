import re
import sys
import os

sys.path.append(os.path.join(os.path.dirname(__file__), '../../../libbeat/tests/system'))

from beat.beat import TestCase

COMMON_FIELDS = ["@timestamp", "beat", "metricset.name", "metricset.host",
                 "metricset.module", "metricset.rtt"]

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class BaseTest(TestCase):

    @classmethod
    def setUpClass(self):
        self.beat_name = "metricbeat"
        self.beat_path = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../"))
        super(BaseTest, self).setUpClass()

    def assert_fields_are_documented(self, evt):
        """
        Assert that all keys present in evt are documented in fields.yml.
        This reads from the global fields.yml, means `make collect` has to be run before the check.
        """
        expected_fields, _ = self.load_fields()
        flat = self.flatten_object(evt, [])

        for key in flat.keys():
            documented = key in expected_fields
            metaKey = key.startswith('@metadata.')
            if not(documented or metaKey):
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

    def assert_no_logged_warnings(self, replace=None):
        """
        Assert that the log file contains no ERROR or WARN lines.
        """
        log = self.get_log()

        pattern = self.build_log_regex("[cfgwarn]")
        log = pattern.sub("", log)

        # Jenkins runs as a Windows service and when Jenkins executes these
        # tests the Beat is confused since it thinks it is running as a service.
        pattern = self.build_log_regex("The service process could not connect to the service controller.")
        log = pattern.sub("", log)

        if replace:
            for r in replace:
                pattern = self.build_log_regex(r)
                log = pattern.sub("", log)
        self.assertNotRegexpMatches(log, "ERROR|WARN")

    def build_log_regex(self, message):
        return re.compile(r"^.*\t(?:ERROR|WARN)\t.*" + message + r".*$", re.MULTILINE)
