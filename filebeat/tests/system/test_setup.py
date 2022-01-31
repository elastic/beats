import unittest
import os
import yaml
from shutil import copytree, copyfile

from filebeat import BaseTest

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class Test(BaseTest):

    def init(self):
        self.elasticsearch_url = self.get_elasticsearch_url()
        print("Using elasticsearch: {}".format(self.elasticsearch_url))
        self.es = self.get_elasticsearch_instance()

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
            elasticsearch=self.get_elasticsearch_template_config(),
        )

        self._setup_dummy_module()

        beat_setup_modules_pipelines = self.start_beat(
            extra_args=[
                "setup",
                "--pipelines",
                "-E", "filebeat.config.modules.path=" + self.working_dir + "/modules.d/*.yml",
            ],
        )
        beat_setup_modules_pipelines.check_wait(exit_code=0)

        version = self.get_beat_version()
        pipeline_name = "filebeat-" + version + "-template-test-module-test-pipeline"
        pipeline = self.es.transport.perform_request("GET", "/_ingest/pipeline/" + pipeline_name)

        assert "date" in pipeline[pipeline_name]["processors"][0]
        assert "remove" in pipeline[pipeline_name]["processors"][1]

    def _setup_dummy_module(self):
        modules_d_path = self.working_dir + "/modules.d"
        modules_path = self.working_dir + "/module"

        for directory in [modules_d_path, modules_path]:
            if not os.path.isdir(directory):
                os.mkdir(directory)

        copytree(self.beat_path + "/tests/system/input/template-test-module", modules_path + "/template-test-module")
        copyfile(
            self.beat_path +
            "/tests/system/input/template-test-module/_meta/config.yml",
            modules_d_path +
            "/test.yml")
