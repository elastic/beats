import jinja2
import os
import sys

sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), '../../../../metricbeat/tests/system')))

from metricbeat import BaseTest as MetricbeatTest


class AuditbeatXPackTest(MetricbeatTest):

    @classmethod
    def setUpClass(self):
        self.beat_name = "auditbeat"
        self.beat_path = os.path.abspath(
            os.path.join(os.path.dirname(__file__), "../../"))

        super(MetricbeatTest, self).setUpClass()

    def setUp(self):
        super(MetricbeatTest, self).setUp()

        # Hack to make jinja2 have the right paths
        self.template_env = jinja2.Environment(
            loader=jinja2.FileSystemLoader([
                os.path.abspath(os.path.join(self.beat_path, "../../auditbeat")),
                os.path.abspath(os.path.join(self.beat_path, "../../libbeat"))
            ])
        )

    # Adapted from metricbeat.py
    def check_metricset(self, module, metricset, fields=[], extras={}, errors_allowed=False, warnings_allowed=False):
        """
        Method to test a metricset for its fields
        """
        # Set to 1 hour so we only test one Fetch
        extras["period"] = "1h"

        self.render_config_template(modules=[{
            "name": module,
            "datasets": [metricset],
            "extras": extras,
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()

        if not warnings_allowed:
            self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertTrue(len(output) >= 1)
        evt = output[0]
        print(evt)

        flattened = self.flatten_object(evt, {})
        for f in fields:
            if not f in flattened:
                raise Exception("Field '{}' not found in event.".format(f))

        # Check for presence of top-level error object.
        if not errors_allowed and "error" in evt:
            raise Exception("Event contains error.")

        self.assert_fields_are_documented(evt)
