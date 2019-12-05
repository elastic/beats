from base import BaseTest
from idxmgmt import IdxMgmt
import os
from nose.plugins.attrib import attr
import unittest
import logging
from nose.tools import raises
from elasticsearch import RequestError

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
        self.index_name = self.alias_name = self.beat_name + "-9.9.9"

        self.custom_alias = self.beat_name + "_foo"
        self.custom_policy = self.beat_name + "_bar"
        self.custom_template = self.beat_name + "_foobar"

        self.es = self.es_client()
        self.idxmgmt = IdxMgmt(self.es, self.index_name)
        self.idxmgmt.delete(indices=[self.custom_alias, self.index_name, self.custom_policy],
                            policies=[self.policy_name, self.custom_policy])

        logging.getLogger("urllib3").setLevel(logging.WARNING)
        logging.getLogger("elasticsearch").setLevel(logging.ERROR)

    def tearDown(self):
        self.idxmgmt.delete(indices=[self.custom_alias, self.index_name, self.custom_policy],
                            policies=[self.policy_name, self.custom_policy])

    def render_config(self, **kwargs):
        self.render_config_template(
            elasticsearch={"hosts": self.get_elasticsearch_url()},
            es_template_name=self.index_name,
            **kwargs
        )

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_setup_default(self):
        """
        Test setup --index-management with default config
        """
        self.render_config()
        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.cmd])

        assert exit_code == 0
        self.idxmgmt.assert_ilm_template_loaded(self.alias_name, self.policy_name, self.alias_name)
        self.idxmgmt.assert_index_template_index_pattern(self.index_name, [self.index_name + "-*"])
        self.idxmgmt.assert_docs_written_to_alias(self.alias_name)
        self.idxmgmt.assert_alias_created(self.alias_name)
        self.idxmgmt.assert_policy_created(self.policy_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    @raises(RequestError)
    def test_setup_default(self):
        """
        Test setup --index-management with default config
        """
        self.render_config()
        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.cmd])

        assert exit_code == 0
        self.idxmgmt.assert_ilm_template_loaded(self.alias_name, self.policy_name, self.alias_name)
        self.idxmgmt.assert_index_template_index_pattern(self.index_name, [self.index_name + "-*"])
        self.idxmgmt.assert_docs_written_to_alias(self.alias_name)
        self.idxmgmt.assert_alias_created(self.alias_name)
        self.idxmgmt.assert_policy_created(self.policy_name)
        # try deleting policy needs to raise an error as it is in use
        self.idxmgmt.delete_policy(self.policy_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_setup_template_disabled(self):
        """
        Test setup --index-management when ilm disabled
        """
        self.render_config()
        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.cmd,
                                              "-E", "setup.template.enabled=false"])

        assert exit_code == 0
        self.idxmgmt.assert_index_template_not_loaded(self.index_name)
        self.idxmgmt.assert_alias_created(self.index_name)
        self.idxmgmt.assert_policy_created(self.policy_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_setup_ilm_disabled(self):
        """
        Test setup --index-management when ilm disabled
        """
        self.render_config()
        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.cmd,
                                              "-E", "setup.ilm.enabled=false"])

        assert exit_code == 0
        self.idxmgmt.assert_index_template_loaded(self.index_name)
        self.idxmgmt.assert_alias_not_created(self.alias_name)
        self.idxmgmt.assert_policy_not_created(self.policy_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_setup_policy_name(self):
        """
        Test  setup --index-management when policy_name is configured
        """
        self.render_config()
        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.cmd,
                                              "-E", "setup.ilm.policy_name=" + self.custom_policy])

        assert exit_code == 0
        self.idxmgmt.assert_ilm_template_loaded(self.alias_name, self.custom_policy, self.alias_name)
        self.idxmgmt.assert_policy_created(self.custom_policy)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
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
                                              "-E", "setup.ilm.policy_name="+policy_name])
        assert exit_code == 0
        resp = self.es.transport.perform_request('GET', '/_ilm/policy/' + policy_name)
        assert "delete" in resp[policy_name]["policy"]["phases"]
        assert "hot" not in resp[policy_name]["policy"]["phases"]

        # ensure ilm policy is overwritten
        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.cmd,
                                              "-E", "setup.ilm.enabled=true",
                                              "-E", "setup.ilm.overwrite=true",
                                              "-E", "setup.ilm.policy_name="+policy_name])
        assert exit_code == 0
        resp = self.es.transport.perform_request('GET', '/_ilm/policy/' + policy_name)
        assert "delete" not in resp[policy_name]["policy"]["phases"]
        assert "hot" in resp[policy_name]["policy"]["phases"]

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_setup_rollover_alias(self):
        """
        Test setup --index-management when ilm.rollover_alias is configured
        """
        self.render_config()
        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.cmd,
                                              "-E", "setup.ilm.rollover_alias=" + self.custom_alias])

        assert exit_code == 0
        self.idxmgmt.assert_ilm_template_loaded(self.custom_alias, self.policy_name, self.custom_alias)
        self.idxmgmt.assert_index_template_index_pattern(self.custom_alias, [self.custom_alias + "-*"])
        self.idxmgmt.assert_docs_written_to_alias(self.custom_alias)
        self.idxmgmt.assert_alias_created(self.custom_alias)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_setup_rollover_alias_with_fieldref(self):
        """
        Test setup --index-management when ilm.rollover_alias is configured and using field reference.
        """
        aliasFieldRef = "%{[agent.name]}-myalias"
        self.render_config()
        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.cmd,
                                              "-E", "setup.ilm.rollover_alias=" + aliasFieldRef])

        self.custom_alias = self.beat_name + "-myalias"

        assert exit_code == 0
        self.idxmgmt.assert_ilm_template_loaded(self.custom_alias, self.policy_name, self.custom_alias)
        self.idxmgmt.assert_index_template_index_pattern(self.custom_alias, [self.custom_alias + "-*"])
        self.idxmgmt.assert_docs_written_to_alias(self.custom_alias)
        self.idxmgmt.assert_alias_created(self.custom_alias)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_setup_template_name_and_pattern(self):
        """
        Test setup --index-management ignores template.name and template.pattern when ilm is enabled
        """
        self.render_config()
        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.cmd,
                                              "-E", "setup.template.name=" + self.custom_template,
                                              "-E", "setup.template.pattern=" + self.custom_template + "*"])

        assert exit_code == 0
        self.idxmgmt.assert_ilm_template_loaded(self.alias_name, self.policy_name, self.alias_name)
        self.idxmgmt.assert_index_template_index_pattern(self.alias_name, [self.alias_name + "-*"])
        self.idxmgmt.assert_docs_written_to_alias(self.alias_name)
        self.idxmgmt.assert_alias_created(self.alias_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
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
        self.idxmgmt.assert_alias_not_created(self.alias_name)
        self.idxmgmt.assert_policy_not_created(self.policy_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
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
        self.idxmgmt.assert_index_template_loaded(self.index_name)

        # check that settings are overwritten
        resp = self.es.transport.perform_request('GET', '/_template/' + self.index_name)
        assert self.index_name in resp
        index = resp[self.index_name]["settings"]["index"]
        assert index["number_of_shards"] == "2", index["number_of_shards"]

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_setup_overwrite_template_on_ilm_policy_created(self):
        """
        Test setup --index-management overwrites template when new ilm policy is created
        """

        # ensure template with ilm rollover_alias name is created, but ilm policy not yet
        self.render_config()
        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.cmd,
                                              "-E", "setup.ilm.enabled=false",
                                              "-E", "setup.template.name=" + self.custom_alias,
                                              "-E", "setup.template.pattern=" + self.custom_alias + "*"])
        assert exit_code == 0
        self.idxmgmt.assert_index_template_loaded(self.custom_alias)
        self.idxmgmt.assert_policy_not_created(self.policy_name)

        # ensure ilm policy is created, triggering overwriting existing template
        exit_code = self.run_beat(extra_args=["setup", self.cmd,
                                              "-E", "setup.template.overwrite=false",
                                              "-E", "setup.template.settings.index.number_of_shards=2",
                                              "-E", "setup.ilm.rollover_alias=" + self.custom_alias])
        assert exit_code == 0
        self.idxmgmt.assert_ilm_template_loaded(self.custom_alias, self.policy_name, self.custom_alias)
        self.idxmgmt.assert_policy_created(self.policy_name)
        # check that template was overwritten
        resp = self.es.transport.perform_request('GET', '/_template/' + self.custom_alias)
        assert self.custom_alias in resp
        index = resp[self.custom_alias]["settings"]["index"]
        assert index["number_of_shards"] == "2", index["number_of_shards"]
