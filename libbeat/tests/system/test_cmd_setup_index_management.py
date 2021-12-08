import logging
import os
import pytest
import unittest

from base import BaseTest
from elasticsearch import RequestError
from idxmgmt import IdxMgmt

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class TestCommandSetupIndexManagement(BaseTest):
    """
    Test beat command `setup` related to ILM policy
    """

    def setUp(self):
        super(TestCommandSetupIndexManagement, self).setUp()

        self.cmd = "--index-management"
        # auto-derived default settings, if nothing else is set
        self.policy_name = self.beat_name
        self.data_stream = self.beat_name + "-9.9.9"

        self.custom_policy = self.beat_name + "_bar"
        self.custom_template = self.beat_name + "_foobar"

        self.es = self.es_client()
        self.idxmgmt = IdxMgmt(self.es, self.data_stream)
        self.idxmgmt.delete(indices=[],
                            policies=[self.policy_name, self.custom_policy],
                            data_streams=[self.data_stream])

        logging.getLogger("urllib3").setLevel(logging.WARNING)
        logging.getLogger("elasticsearch").setLevel(logging.ERROR)

    def tearDown(self):
        self.idxmgmt.delete(indices=[],
                            policies=[self.policy_name, self.custom_policy],
                            data_streams=[self.data_stream])

    def render_config(self, **kwargs):
        self.render_config_template(
            elasticsearch={"hosts": self.get_elasticsearch_url()},
            es_template_name=self.data_stream,
            **kwargs
        )

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_setup_default(self):
        """
        Test setup --index-management with default config
        """
        self.render_config()
        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.cmd])

        assert exit_code == 0
        self.idxmgmt.assert_index_template_loaded(self.data_stream)
        self.idxmgmt.assert_index_template_index_pattern(self.data_stream, [self.data_stream])
        self.idxmgmt.assert_policy_created(self.policy_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_setup_template_disabled(self):
        """
        Test setup --index-management when ilm disabled
        """

        self.render_config()
        exit_code = self.run_beat(logging_args=["-v", "-e", "-d", "*"],
                                  extra_args=["setup", self.cmd,
                                              "-E", "setup.template.enabled=false"])

        assert exit_code == 0
        self.idxmgmt.assert_index_template_not_loaded(self.data_stream+"ba")
        self.idxmgmt.assert_policy_created(self.policy_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_setup_ilm_disabled(self):
        """
        Test setup --index-management when ilm disabled
        """
        self.render_config()
        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.cmd,
                                              "-E", "setup.ilm.enabled=false"])

        assert exit_code == 0
        self.idxmgmt.assert_index_template_loaded(self.data_stream)
        self.idxmgmt.assert_policy_not_created(self.policy_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_setup_policy_name(self):
        """
        Test  setup --index-management when policy_name is configured
        """
        self.render_config()
        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.cmd,
                                              "-E", "setup.ilm.policy_name=" + self.custom_policy])

        assert exit_code == 0
        self.idxmgmt.assert_index_template_loaded(self.data_stream)
        self.idxmgmt.assert_policy_created(self.custom_policy)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_setup_ilm_policy_no_overwrite(self):
        """
        Test setup --index-management respects overwrite configuration
        """
        policy_name = "mockbeat-test"
        # update policy to verify overwrite behaviour
        self.es.transport.perform_request('PUT', '/_ilm/policy/' + policy_name,
                                          body={
                                              "policy": {
                                                 "phases": {
                                                     "delete": {
                                                         "actions": {
                                                             "delete": {}
                                                         }
                                                     }
                                                 }
                                              }
                                          })
        resp = self.es.transport.perform_request('GET', '/_ilm/policy/' + policy_name)
        assert "delete" in resp[policy_name]["policy"]["phases"]
        assert "hot" not in resp[policy_name]["policy"]["phases"]

        # ensure ilm policy is not overwritten
        self.render_config()
        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.cmd,
                                              "-E", "setup.ilm.enabled=true",
                                              "-E", "setup.ilm.overwrite=false",
                                              "-E", "setup.ilm.policy_name=" + policy_name])
        assert exit_code == 0
        resp = self.es.transport.perform_request('GET', '/_ilm/policy/' + policy_name)
        assert "delete" in resp[policy_name]["policy"]["phases"]
        assert "hot" not in resp[policy_name]["policy"]["phases"]

        # ensure ilm policy is overwritten
        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.cmd,
                                              "-E", "setup.ilm.enabled=true",
                                              "-E", "setup.ilm.overwrite=true",
                                              "-E", "setup.ilm.policy_name=" + policy_name])
        assert exit_code == 0
        resp = self.es.transport.perform_request('GET', '/_ilm/policy/' + policy_name)
        assert "delete" not in resp[policy_name]["policy"]["phases"]
        assert "hot" in resp[policy_name]["policy"]["phases"]

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_setup_template_name_and_pattern_on_ilm_disabled(self):
        """
        Test setup --index-management respects template.name and template.pattern when ilm is disabled
        """
        self.render_config()
        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.cmd,
                                              "-E", "setup.ilm.enabled=false",
                                              "-E", "setup.template.name=" + self.custom_template,
                                              "-E", "setup.template.pattern=" + self.custom_template + "*"])

        assert exit_code == 0
        self.idxmgmt.assert_index_template_loaded(self.custom_template)
        self.idxmgmt.assert_index_template_index_pattern(self.custom_template, [self.custom_template + "*"])
        self.idxmgmt.assert_policy_not_created(self.policy_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_setup_template_with_opts(self):
        """
        Test setup --index-management with config options
        """
        self.render_config()

        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.cmd,
                                              "-E", "setup.ilm.enabled=false",
                                              "-E", "setup.template.settings.index.number_of_shards=2"])

        assert exit_code == 0
        self.idxmgmt.assert_index_template_loaded(self.data_stream)

        # check that settings are overwritten
        resp = self.es.transport.perform_request('GET', '/_index_template/' + self.data_stream)
        found = False
        for index_template in resp["index_templates"]:
            if self.data_stream == index_template["name"]:
                found = True
                index = index_template["index_template"]["template"]["settings"]["index"]
                assert index["number_of_shards"] == "2", index["number_of_shards"]
        assert found

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_setup_overwrite_template_on_ilm_policy_created(self):
        """
        Test setup --index-management overwrites template when new ilm policy is created
        """

        # ensure template with ilm rollover_alias name is created, but ilm policy not yet
        self.render_config()
        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.cmd,
                                              "-E", "setup.ilm.enabled=false",
                                              "-E", "setup.template.priority=160",
                                              "-E", "setup.template.name=" + self.custom_template,
                                              "-E", "setup.template.pattern=" + self.custom_template + "*"])
        assert exit_code == 0
        self.idxmgmt.assert_index_template_loaded(self.custom_template)
        self.idxmgmt.assert_policy_not_created(self.policy_name)

        # ensure ilm policy is created, triggering overwriting existing template
        exit_code = self.run_beat(extra_args=["setup", "-d", "*", self.cmd,
                                              "-E", "setup.template.overwrite=true",
                                              "-E", "setup.template.name=" + self.custom_template,
                                              "-E", "setup.template.pattern=" + self.custom_template + "*",
                                              "-E", "setup.template.settings.index.number_of_shards=2"])
        assert exit_code == 0
        self.idxmgmt.assert_index_template_loaded(self.custom_template)
        self.idxmgmt.assert_policy_created(self.policy_name)
        # check that template was overwritten
        resp = self.es.transport.perform_request('GET', '/_index_template/' + self.custom_template)

        found = False
        for index_template in resp["index_templates"]:
            if index_template["name"] == self.custom_template:
                found = True
                index = index_template["index_template"]["template"]["settings"]["index"]
                assert index["number_of_shards"] == "2", index["number_of_shards"]
        assert found
