import os
import metricbeat
import unittest
from nose.plugins.attrib import attr


class TestFoo(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['foo']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_status(self):
        """
        MySQL module outputs an event.
        """
        self.render_config_template(modules=[{
            "name": "foo",
            "metricsets": ["bar"],
            "hosts": "localhost:8080",
            "period": "5s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), 1)
        evt = output[0]

        print(evt)

        # self.assertItemsEqual(self.de_dot(MYSQL_FIELDS), evt.keys())
        # mysql_info = evt["mysql"]["status"]

        self.assert_fields_are_documented(evt)
