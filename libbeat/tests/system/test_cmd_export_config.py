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
        self.render_config_template(self.beat_name, self.output, metrics_period='1234')
        exit_code = self.run_beat(extra_args=["export", "config"], config=self.config)

        assert exit_code == 0
        assert self.log_contains("filename: mockbeat")
        assert self.log_contains("period: 1234")

    def test_config_environment_variable(self):
        """
        Test export config works but doesn"t expose environment variable.
        """
        self.render_config_template(self.beat_name, self.output,
                                    metrics_period="${METRIC_PERIOD}")
        exit_code = self.run_beat(extra_args=["export", "config"], config=self.config,
                                  env={'METRIC_PERIOD': '1234'})

        assert exit_code == 0
        assert self.log_contains("filename: mockbeat")
        assert self.log_contains("period: ${METRIC_PERIOD}")
