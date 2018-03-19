import os
import unittest
from filebeat import BaseTest
from elasticsearch import Elasticsearch
from beat.beat import INTEGRATION_TESTS


class Test(BaseTest):

    def test_base(self):
        """
        Test if the basic fields exist.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
        )

        with open(self.working_dir + "/test.log", "w") as f:
            f.write("test message\n")

        filebeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=1))
        filebeat.check_kill_and_wait()

        output = self.read_output()[0]
        assert "@timestamp" in output
        assert "prospector.type" in output
        assert "input.type" in output

    def test_invalid_config_with_removed_settings(self):
        """
        Checks if filebeat fails to load if removed settings have been used:
        """
        self.render_config_template()

        exit_code = self.run_beat(extra_args=[
            "-E", "filebeat.spool_size=2048",
            "-E", "filebeat.publish_async=true",
            "-E", "filebeat.idle_timeout=1s",
        ])

        assert exit_code == 1
        assert self.log_contains("setting 'filebeat.spool_size'"
                                 " has been removed")
        assert self.log_contains("setting 'filebeat.publish_async'"
                                 " has been removed")
        assert self.log_contains("setting 'filebeat.idle_timeout'"
                                 " has been removed")

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    def test_template(self):
        """
        Test that the template can be loaded with `setup --template`
        """
        es = Elasticsearch([self.get_elasticsearch_url()])
        self.render_config_template(
            elasticsearch={"host": self.get_elasticsearch_url()},
        )
        exit_code = self.run_beat(extra_args=["setup", "--template"])

        assert exit_code == 0
        assert self.log_contains('Loaded index template')
        assert len(es.cat.templates(name='filebeat-*', h='name')) > 0
