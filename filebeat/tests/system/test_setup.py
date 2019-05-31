import unittest
import os
import yaml
from shutil import copyfile

from elasticsearch import Elasticsearch

from filebeat import BaseTest

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class Test(BaseTest):

    def init(self):
        self.elasticsearch_url = self.get_elasticsearch_url()
        print("Using elasticsearch: {}".format(self.elasticsearch_url))
        self.es = Elasticsearch([self.elasticsearch_url])

    @unittest.skipIf(not INTEGRATION_TESTS,
                     "integration tests are disabled, run with INTEGRATION_TESTS=1 to enable them.")
    @unittest.skipIf(os.getenv("TESTING_ENVIRONMENT") == "2x",
                     "integration test not available on 2.x")
    def test_setup_modules_d_config(self):
        """
        Check if template settings are applied to Ingest pipelines when configured from modules.d.
        """
        self.init()
        self.render_config_template(
            modules=True,
            elasticsearch={
                "host": self.get_elasticsearch_url(),
            },
        )

        modules_d_path = self.working_dir + "/modules.d"
        if not os.path.isdir(modules_d_path):
            os.mkdir(modules_d_path)
        copyfile(self.beat_path + "/tests/system/input/system.yml", modules_d_path + "/system.yml")

        beat_setup_modules_pipelines = self.start_beat(
            extra_args=[
                "setup",
                "--pipelines",
                "-path.home=" + self.beat_path,
                "-E", "filebeat.config.modules.path=" + modules_d_path + "/*.yml",
            ],
            configure_home=False,
        )
        beat_setup_modules_pipelines.check_wait(exit_code=0)

        version = self.get_beat_version()
        system_syslog_pipeline_name = "filebeat-" + version + "-system-syslog-pipeline"
        system_syslog_pipeline = self.es.transport.perform_request("GET",
                                                                   "/_ingest/pipeline/" + system_syslog_pipeline_name)

        assert "timezone" in system_syslog_pipeline[system_syslog_pipeline_name]["processors"][4]["date"]

        system_auth_pipeline_name = "filebeat-" + version + "-system-auth-pipeline"
        system_auth_pipeline = self.es.transport.perform_request("GET",
                                                                 "/_ingest/pipeline/" + system_auth_pipeline_name)

        assert "timezone" not in system_auth_pipeline[system_auth_pipeline_name]["processors"][4]["date"]
