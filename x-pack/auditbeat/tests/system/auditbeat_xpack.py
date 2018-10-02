import jinja2
import os

from auditbeat import BaseTest as AuditbeatTest


class AuditbeatXPackTest(AuditbeatTest):

    @classmethod
    def setUpClass(self):
        self.beat_name = "auditbeat"
        self.beat_path = os.path.abspath(
            os.path.join(os.path.dirname(__file__), "../../"))

        super(AuditbeatTest, self).setUpClass()

    def setUp(self):
        super(AuditbeatTest, self).setUp()

        # Hack to make jinja2 have the right paths
        self.template_env = jinja2.Environment(
            loader=jinja2.FileSystemLoader([
                os.path.abspath(os.path.join(self.beat_path, "../../auditbeat")),
                os.path.abspath(os.path.join(self.beat_path, "../../libbeat"))
            ])
        )

    # Adapted from metricbeat.py
    def check_metricset(self, module, metricset, fields=[], warnings_allowed=False):
        """
        Method to test a metricset for its fields
        """
        self.render_config_template(modules=[{
            "name": module,
            "metricsets": [metricset],
            "period": "10s",
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

        self.assert_fields_are_documented(evt)
