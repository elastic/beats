from base import BaseTest
from nose.plugins.attrib import attr
from elasticsearch import Elasticsearch, TransportError

import logging
import os
import shutil
import unittest


INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class TestCommands(BaseTest):
    """
    Test beat subcommands
    """

    def setUp(self):
        super(BaseTest, self).setUp()
        shutil.copy(self.beat_path + "/_meta/config.yml",
                    os.path.join(self.working_dir, "libbeat.yml"))

    def test_migration_default(self):
        """
        Tests that if no migration flag is set, no alias exists. By default migratin is off.
        """

        exit_code = self.run_beat(
            extra_args=[
                "export", "template",
                "-E", "setup.template.fields=" + os.path.join(self.working_dir, "fields.yml"),
            ],
            config="libbeat.yml")

        assert exit_code == 0
        assert self.log_contains_count('"type": "alias"') == 0

    def test_migration_false(self):
        """
        If migration flag is set to false, no alias exist
        """

        exit_code = self.run_beat(
            extra_args=[
                "export", "template",
                "-E", "setup.template.fields=" + os.path.join(self.working_dir, "fields.yml"),
                "-E", "migration.enabled=true",
            ],
            config="libbeat.yml")

        assert exit_code == 0
        assert self.log_contains('"type": "alias"')

    def test_migration_true(self):
        """
        Test that if migration flag is set to true, some alias are loaded.
        """

        exit_code = self.run_beat(
            extra_args=[
                "export", "template",
                "-E", "setup.template.fields=" + os.path.join(self.working_dir, "fields.yml"),
                "-E", "migration.enabled=false",
            ],
            config="libbeat.yml")

        assert exit_code == 0
        assert self.log_contains_count('"type": "alias"') == 0
