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
        self.fields_path = os.path.join(self.beat_path, "template/testdata/fields.yml")

    def test_migration_default(self):
        """
        If no migration flag is set, no migration alias exists. By default migration is off.
        """

        exit_code = self.run_beat(
            extra_args=[
                "export", "template",
                "-E", "setup.template.fields=" + self.fields_path,
            ],
            config="libbeat.yml")

        assert exit_code == 0
        assert self.log_contains('migration_alias_false')
        assert not self.log_contains('migration_alias_true')

    def test_migration_false(self):
        """
        If migration flag is set to false, no migration alias exist
        """

        exit_code = self.run_beat(
            extra_args=[
                "export", "template",
                "-E", "setup.template.fields=" + self.fields_path,
                "-E", "migration.6_to_7.enabled=false",
            ],
            config="libbeat.yml")

        assert exit_code == 0
        assert self.log_contains('migration_alias_false')
        assert not self.log_contains('migration_alias_true')

    def test_migration_true(self):
        """
        If migration flag is set to true, migration alias are loaded.
        """

        exit_code = self.run_beat(
            extra_args=[
                "export", "template",
                "-E", "setup.template.fields=" + self.fields_path,
                "-E", "migration.6_to_7.enabled=true",
            ],
            config="libbeat.yml")

        assert exit_code == 0
        assert self.log_contains('migration_alias_false')
        assert self.log_contains('migration_alias_true')
