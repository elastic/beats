import os
from metricbeat import BaseTest
import unittest
from nose.plugins.attrib import attr
import urllib2
import time


class ConfigTest(BaseTest):
    def test_service_name(self):
        """
        Test setting service name
        """
        service_name = "testing"
        self.render_config_template(modules=[{
            # Do it with any module that don't set the service name by itself
            "name": "system",
            "metricsets": ["cpu"],
            "period": "5s",
            "extras": {
                "service.name": service_name,
            },
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        event = output[0]
        self.assert_fields_are_documented(event)

        self.assertEqual(service_name, event["service"]["name"])
