import re
import sys
import os
import shutil
import unittest
from auditbeat import BaseTest
from elasticsearch import Elasticsearch
from beat.beat import INTEGRATION_TESTS


class Test(BaseTest):
    def test_start_stop(self):
        """
        Auditbeat starts and stops without error.
        """
        self.render_config_template(
            modules=[{
                "name": "file_integrity",
                "extras": {
                    "paths": ["file.example"],
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
    def test_template(self):
        """
        Test that the template can be loaded with `setup --template`
        """
        es = Elasticsearch([self.get_elasticsearch_url()])

        self.render_config_template(
            modules=[{
                "name": "file_integrity",
                "extras": {
                    "paths": ["file.example"],
                }
            }],
            elasticsearch={"host": self.get_elasticsearch_url()})
        exit_code = self.run_beat(extra_args=["setup", "--template"])

        assert exit_code == 0
        assert self.log_contains('Loaded index template')
        assert len(es.cat.templates(name='auditbeat-*', h='name')) > 0

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    def test_dashboards(self):
        """
        Test that the dashboards can be loaded with `setup --dashboards`
        """

        kibana_dir = os.path.join(self.beat_path, "_meta", "kibana")
        shutil.copytree(kibana_dir, os.path.join(self.working_dir, "kibana"))

        es = Elasticsearch([self.get_elasticsearch_url()])
        self.render_config_template(
            modules=[{
                "name": "file_integrity",
                "extras": {
                    "paths": ["file.example"],
                }
            }],
            elasticsearch={"host": self.get_elasticsearch_url()},
            kibana={"host": self.get_kibana_url()},
        )
        exit_code = self.run_beat(extra_args=["setup", "--dashboards"])

        assert exit_code == 0
        assert self.log_contains("Kibana dashboards successfully loaded.")
