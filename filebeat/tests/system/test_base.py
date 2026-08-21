import os
import unittest
from filebeat import BaseTest
from beat.beat import INTEGRATION_TESTS
from beat import common_tests


class Test(BaseTest, common_tests.TestExportsMixin, common_tests.TestDashboardMixin):

    def setUp(self):
        super(Test, self).setUp()
        self.render_config_template(
            elasticsearch=self.get_elasticsearch_template_config(),
        )
        self.es = self.get_elasticsearch_instance()

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    def test_index_management(self):
        """
        Test that the template can be loaded with `setup --index-management`
        """
        self.render_config_template(
            elasticsearch=self.get_elasticsearch_template_config(),
        )
        exit_code = self.run_beat(extra_args=["setup", "--index-management"])

        assert exit_code == 0
        assert self.log_contains('Loaded index template')
        assert len(self.es.cat.templates(name='filebeat-*', h='name')) > 0
