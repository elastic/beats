import os
import unittest
from filebeat import BaseTest


class Test(BaseTest):

    def test_export_index_pattern(self):
        """
        Test export index pattern
        """
        self.render_config_template()
        exit_code = self.run_beat(
            logging_args=[],
            extra_args=["export", "index-pattern"])

        assert exit_code == 0
        assert self.log_contains('"type": "index-pattern"')
        assert self.log_contains('beat.name') == False

    def test_export_index_pattern_migration(self):
        """
        Test export index pattern with migration flag enabled
        """
        self.render_config_template()
        exit_code = self.run_beat(
            logging_args=[],
            extra_args=["export", "index-pattern", "-E", "migration.6_to_7.enabled:true"])

        assert exit_code == 0
        assert self.log_contains('"type": "index-pattern"')
        assert self.log_contains('beat.name')
