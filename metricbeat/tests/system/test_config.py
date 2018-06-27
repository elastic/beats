import os
import metricbeat
import unittest
from nose.plugins.attrib import attr
import urllib2
import time


class ConfigTest(metricbeat.BaseTest):

    def test_invalid_config_with_removed_settings(self):
        """
        Checks if metricbeat fails to load a module if remove settings have been used:
        """
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["cpu"],
            "period": "5s",
        }])

        exit_code = self.run_beat(extra_args=[
            "-E",
            "metricbeat.modules.0.filters.0.include_fields='field1,field2'"
        ])
        assert exit_code == 1
        assert self.log_contains("setting 'metricbeat.modules.0.filters'"
                                 " has been removed")

    def get_host(self):
        return 'http://' + os.getenv('ES_HOST', 'localhost') + ':' + os.getenv('ES_PORT', '9200')
