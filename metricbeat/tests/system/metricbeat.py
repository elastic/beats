import re
import sys
import os

sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), '../../../libbeat/tests/system')))

from beat.beat import TestCase

COMMON_FIELDS = ["@timestamp", "agent", "metricset.name", "metricset.host",
                 "metricset.module", "metricset.rtt", "host.name", "service.name", "event", "ecs"]

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)

import logging
logging.getLogger("urllib3").setLevel(logging.WARNING)


class BaseTest(TestCase):

    @classmethod
    def setUpClass(self):
        if not hasattr(self, 'beat_name'):
            self.beat_name = "metricbeat"

        if not hasattr(self, 'beat_path'):
            self.beat_path = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../"))

        super(BaseTest, self).setUpClass()

    def setUp(self):
        super(BaseTest, self).setUp()

    def tearDown(self):
        super(BaseTest, self).tearDown()

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

        pattern = self.build_log_regex("\[cfgwarn\]")
        log = pattern.sub("", log)

        # Jenkins runs as a Windows service and when Jenkins executes these
        # tests the Beat is confused since it thinks it is running as a service.
        pattern = self.build_log_regex("The service process could not connect to the service controller.")
        log = pattern.sub("", log)

        if replace:
            for r in replace:
                pattern = self.build_log_regex(r)
                log = pattern.sub("", log)
        self.assertNotRegexpMatches(log, "\tERROR\t|\tWARN\t")

    def build_log_regex(self, message):
        return re.compile(r"^.*\t(?:ERROR|WARN)\t.*" + message + r".*$", re.MULTILINE)

    def check_metricset(self, module, metricset, hosts, fields=[], extras=[]):
        """
        Method to test a metricset for its fields
        """
        self.render_config_template(modules=[{
            "name": module,
            "metricsets": [metricset],
            "hosts": hosts,
            "period": "1s",
            "extras": extras,
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        print output
        self.assertTrue(len(output) >= 1)
        evt = output[0]
        print(evt)

        fields = COMMON_FIELDS + fields
        print fields
        self.assertItemsEqual(self.de_dot(fields), evt.keys())

        self.assert_fields_are_documented(evt)
