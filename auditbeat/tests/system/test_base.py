import re
import sys
import os
import shutil
import unittest
from auditbeat import *
from elasticsearch import Elasticsearch
from beat.beat import INTEGRATION_TESTS
from beat import common_tests


class Test(BaseTest, common_tests.TestExportsMixin, common_tests.TestDashboardMixin):
    def test_start_stop(self):
        """
        Auditbeat starts and stops without error.
        """
        dirs = [self.temp_dir("auditbeat_test")]
        with PathCleanup(dirs):
            self.render_config_template(
                modules=[{
                    "name": "file_integrity",
                    "extras": {
                        "paths": dirs,
                    }
                }],
            )
            proc = self.start_beat()
            self.wait_until(lambda: self.log_contains("start running"))
            proc.check_kill_and_wait()
            self.assert_no_logged_warnings()

            # Ensure all Beater stages are used.
            assert self.log_contains("Setup Beat: auditbeat")
            assert self.log_contains("auditbeat start running")
            assert self.log_contains("auditbeat stopped")

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    def test_index_management(self):
        """
        Test that the template can be loaded with `setup --index-management`
        """
        dirs = [self.temp_dir("auditbeat_test")]
        with PathCleanup(dirs):
            es = Elasticsearch([self.get_elasticsearch_url()])

            self.render_config_template(
                modules=[{
                    "name": "file_integrity",
                    "extras": {
                        "paths": dirs,
                    }
                }],
                elasticsearch={"host": self.get_elasticsearch_url()})
            self.run_beat(extra_args=["setup", "--index-management"], exit_code=0)

            assert self.log_contains('Loaded index template')
            assert len(es.cat.templates(name='auditbeat-*', h='name')) > 0

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    def test_dashboards(self):
        """
        Test that the dashboards can be loaded with `setup --dashboards`
        """

        dirs = [self.temp_dir("auditbeat_test")]
        with PathCleanup(dirs):
            kibana_dir = os.path.join(self.beat_path, "build", "kibana")
            shutil.copytree(kibana_dir, os.path.join(self.working_dir, "kibana"))

            es = Elasticsearch([self.get_elasticsearch_url()])
            self.render_config_template(
                modules=[{
                    "name": "file_integrity",
                    "extras": {
                        "paths": dirs,
                    }
                }],
                elasticsearch={"host": self.get_elasticsearch_url()},
                kibana={"host": self.get_kibana_url()},
            )
            self.run_beat(extra_args=["setup", "--dashboards"], exit_code=0)

            assert self.log_contains("Kibana dashboards successfully loaded.")
