import os
from base import BaseTest

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class TestCommandExportConfig(BaseTest):
    """
    Test beat command `export config`
    """

    def setUp(self):
        super(TestCommandExportConfig, self).setUp()

        self.config = "libbeat.yml"
        self.output = os.path.join(self.working_dir, self.config)

    def test_default(self):
        """
        Test export config works
        """
        self.render_config_template(self.beat_name, self.output, file_name='some-file')
        exit_code = self.run_beat(extra_args=["export", "config"], config=self.config)

        assert exit_code == 0
        assert self.log_contains("filename: mockbeat")
        assert self.log_contains("name: some-file")

    def test_config_environment_variable(self):
        """
        Test export config works but doesn"t expose environment variable.
        """
        self.render_config_template(self.beat_name, self.output,
                                    file_name="${FILE_NAME}")
        exit_code = self.run_beat(extra_args=["export", "config"], config=self.config,
                                  env={'FILE_NAME': 'some-file'})

        assert exit_code == 0
        assert self.log_contains("filename: mockbeat")
        assert self.log_contains("name: ${FILE_NAME}")
