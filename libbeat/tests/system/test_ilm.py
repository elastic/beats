from base import BaseTest
from idxmgmt import IdxMgmt
import os
from nose.plugins.attrib import attr
import unittest
import shutil
import datetime
import logging
import json

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class TestRunILM(BaseTest):

    def setUp(self):
        super(TestRunILM, self).setUp()

        self.alias_name = self.policy_name = self.index_name = self.beat_name + "-9.9.9"
        self.custom_alias = self.beat_name + "_foo"
        self.custom_policy = self.beat_name + "_bar"
        self.es = self.es_client()
        self.idxmgmt = IdxMgmt(self.es, self.index_name)
        self.idxmgmt.delete(indices=[self.custom_alias, self.index_name, self.custom_policy])

    def tearDown(self):
        self.idxmgmt.delete(indices=[self.custom_alias, self.index_name, self.custom_policy])

    def render_config(self, **kwargs):
        self.render_config_template(
            elasticsearch={"hosts": self.get_elasticsearch_url()},
            es_template_name=self.index_name,
            **kwargs
        )

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_ilm_default(self):
        """
        Test ilm default settings to load ilm policy, write alias and ilm template
        """
        self.render_config()
        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains("ILM policy successfully loaded"))
        self.wait_until(lambda: self.log_contains("PublishEvents: 1 events have been published"))
        proc.check_kill_and_wait()

        self.idxmgmt.assert_ilm_template_loaded(self.alias_name, self.policy_name, self.alias_name)
        self.idxmgmt.assert_alias_created(self.alias_name)
        self.idxmgmt.assert_policy_created(self.policy_name)
        self.idxmgmt.assert_docs_written_to_alias(self.alias_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_ilm_disabled(self):
        """
        Test ilm disabled to not load ilm related components
        """

        self.render_config(ilm={"enabled": False})
        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains("PublishEvents: 1 events have been published"))
        proc.check_kill_and_wait()

        self.idxmgmt.assert_index_template_loaded(self.index_name)
        self.idxmgmt.assert_alias_not_created(self.alias_name)
        self.idxmgmt.assert_policy_not_created(self.policy_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_policy_name(self):
        """
        Test setting ilm policy name
        """

        policy_name = self.beat_name + "_foo"
        self.render_config(ilm={"enabled": True, "policy_name": policy_name})

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains("ILM policy successfully loaded"))
        self.wait_until(lambda: self.log_contains("PublishEvents: 1 events have been published"))
        proc.check_kill_and_wait()

        self.idxmgmt.assert_ilm_template_loaded(self.alias_name, policy_name, self.alias_name)
        self.idxmgmt.assert_docs_written_to_alias(self.alias_name)
        self.idxmgmt.assert_policy_created(policy_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_rollover_alias(self):
        """
        Test settings ilm rollover alias
        """

        self.render_config(ilm={"enabled": True, "rollover_alias": self.custom_alias})

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains("ILM policy successfully loaded"))
        self.wait_until(lambda: self.log_contains("PublishEvents: 1 events have been published"))
        proc.check_kill_and_wait()

        self.idxmgmt.assert_ilm_template_loaded(self.custom_alias, self.policy_name, self.custom_alias)
        self.idxmgmt.assert_docs_written_to_alias(self.custom_alias)
        self.idxmgmt.assert_alias_created(self.custom_alias)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_pattern(self):
        """
        Test setting ilm pattern
        """

        pattern = "1"
        self.render_config(ilm={"enabled": True, "pattern": pattern})

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains("ILM policy successfully loaded"))
        self.wait_until(lambda: self.log_contains("PublishEvents: 1 events have been published"))
        proc.check_kill_and_wait()

        self.idxmgmt.assert_ilm_template_loaded(self.alias_name, self.policy_name, self.alias_name)
        self.idxmgmt.assert_alias_created(self.alias_name, pattern=pattern)
        self.idxmgmt.assert_docs_written_to_alias(self.alias_name, pattern=pattern)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_pattern_date(self):
        """
        Test setting ilm pattern with date
        """

        pattern = "'{now/d}'"
        self.render_config(ilm={"enabled": True, "pattern": pattern})

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains("ILM policy successfully loaded"))
        self.wait_until(lambda: self.log_contains("PublishEvents: 1 events have been published"))
        proc.check_kill_and_wait()

        resolved_pattern = datetime.datetime.now().strftime("%Y.%m.%d")

        self.idxmgmt.assert_ilm_template_loaded(self.alias_name, self.policy_name, self.alias_name)
        self.idxmgmt.assert_alias_created(self.alias_name, pattern=resolved_pattern)
        self.idxmgmt.assert_docs_written_to_alias(self.alias_name, pattern=resolved_pattern)


class TestCommandSetupILMPolicy(BaseTest):
    """
    Test beat command `setup` related to ILM policy
    """

    def setUp(self):
        super(TestCommandSetupILMPolicy, self).setUp()

        self.setupCmd = "--ilm-policy"

        self.alias_name = self.policy_name = self.index_name = self.beat_name + "-9.9.9"
        self.custom_alias = self.beat_name + "_foo"
        self.custom_policy = self.beat_name + "_bar"
        self.es = self.es_client()
        self.idxmgmt = IdxMgmt(self.es, self.index_name)
        self.idxmgmt.delete(indices=[self.custom_alias, self.index_name, self.custom_policy])

        logging.getLogger("urllib3").setLevel(logging.WARNING)
        logging.getLogger("elasticsearch").setLevel(logging.ERROR)

    def tearDown(self):
        self.idxmgmt.delete(indices=[self.custom_alias, self.index_name, self.custom_policy])

    def render_config(self, **kwargs):
        self.render_config_template(
            elasticsearch={"hosts": self.get_elasticsearch_url()},
            es_template_name=self.index_name,
            **kwargs
        )

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_setup_ilm_policy_and_template(self):
        """
        Test combination of ilm policy and template setup
        """
        self.render_config()

        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.setupCmd, "--template"])

        assert exit_code == 0
        self.idxmgmt.assert_ilm_template_loaded(self.alias_name, self.policy_name, self.alias_name)
        self.idxmgmt.assert_docs_written_to_alias(self.alias_name)
        self.idxmgmt.assert_alias_created(self.alias_name)
        self.idxmgmt.assert_policy_created(self.policy_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_setup_ilm_default(self):
        """
        Test ilm policy setup with default config
        """
        self.render_config()

        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.setupCmd])

        assert exit_code == 0
        self.idxmgmt.assert_ilm_template_loaded(self.alias_name, self.policy_name, self.alias_name)
        self.idxmgmt.assert_index_template_index_pattern(self.alias_name, [self.alias_name + "-*"])
        self.idxmgmt.assert_docs_written_to_alias(self.alias_name)
        self.idxmgmt.assert_alias_created(self.alias_name)
        self.idxmgmt.assert_policy_created(self.policy_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_setup_ilm_disabled(self):
        """
        Test ilm policy setup when ilm disabled
        """
        self.render_config()

        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.setupCmd,
                                              "-E", "setup.ilm.enabled=false"])

        assert exit_code == 0
        self.idxmgmt.assert_index_template_loaded(self.index_name)
        self.idxmgmt.assert_alias_not_created(self.alias_name)
        self.idxmgmt.assert_policy_not_created(self.policy_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_policy_name(self):
        """
        Test ilm policy setup when policy_name is configured
        """
        self.render_config()

        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.setupCmd,
                                              "-E", "setup.ilm.policy_name=" + self.custom_policy])

        assert exit_code == 0
        self.idxmgmt.assert_ilm_template_loaded(self.alias_name, self.custom_policy, self.alias_name)
        self.idxmgmt.assert_policy_created(self.custom_policy)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_rollover_alias(self):
        """
        Test ilm policy setup when rollover_alias is configured
        """
        self.render_config()

        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.setupCmd,
                                              "-E", "setup.ilm.rollover_alias=" + self.custom_alias])

        assert exit_code == 0
        self.idxmgmt.assert_ilm_template_loaded(self.custom_alias, self.policy_name, self.custom_alias)
        self.idxmgmt.assert_docs_written_to_alias(self.custom_alias)
        self.idxmgmt.assert_alias_created(self.custom_alias)


class TestCommandExportILMPolicy(BaseTest):
    """
    Test beat command `export ilm-policy`
    """

    def setUp(self):
        super(TestCommandExportILMPolicy, self).setUp()

        self.config = "libbeat.yml"
        self.output = os.path.join(self.working_dir, self.config)
        shutil.copy(os.path.join(self.beat_path, "fields.yml"), self.output)
        self.policy_name = self.beat_name + "-9.9.9"
        self.cmd = "ilm-policy"

    def assert_log_contains_policy(self):
        assert self.log_contains('ILM policy successfully loaded.')
        assert self.log_contains('"max_age": "30d"')
        assert self.log_contains('"max_size": "50gb"')

    def assert_log_contains_write_alias(self):
        assert self.log_contains('Write alias successfully generated.')

    def test_default(self):
        """
        Test ilm-policy export with default config
        """

        exit_code = self.run_beat(extra_args=["export", self.cmd],
                                  config=self.config)

        assert exit_code == 0
        self.assert_log_contains_policy()
        self.assert_log_contains_write_alias()

    def test_load_disabled(self):
        """
        Test ilm-policy export when ilm disabled in config
        """

        exit_code = self.run_beat(extra_args=["export", self.cmd, "-E", "setup.ilm.enabled=false"],
                                  config=self.config)

        assert exit_code == 0
        self.assert_log_contains_policy()
        self.assert_log_contains_write_alias()

    def test_changed_policy_name(self):
        """
        Test ilm-policy export when policy name is changed
        """
        policy_name = "foo"

        exit_code = self.run_beat(extra_args=["export", self.cmd, "-E", "setup.ilm.policy_name=" + policy_name],
                                  config=self.config)

        assert exit_code == 0
        self.assert_log_contains_policy()
        self.assert_log_contains_write_alias()

    def test_export_to_file_absolute_path(self):
        """
        Test export ilm policy to file with absolute file path
        """
        base_path = os.path.abspath(os.path.join(self.beat_path, os.path.dirname(__file__), "export"))
        exit_code = self.run_beat(
            extra_args=["export", self.cmd, "--dir=" + base_path],
            config=self.config)

        assert exit_code == 0

        file = os.path.join(base_path, "policy", self.policy_name + '.json')
        with open(file) as f:
            policy = json.load(f)
        assert policy["policy"]["phases"]["hot"]["actions"]["rollover"]["max_size"] == "50gb", policy
        assert policy["policy"]["phases"]["hot"]["actions"]["rollover"]["max_age"] == "30d", policy

        os.remove(file)

    def test_export_to_file_relative_path(self):
        """
        Test export ilm policy to file with relative file path
        """
        path = os.path.join(os.path.dirname(__file__), "export")
        exit_code = self.run_beat(
            extra_args=["export", self.cmd, "--dir=" + path],
            config=self.config)

        assert exit_code == 0

        base_path = os.path.abspath(os.path.join(self.beat_path, os.path.dirname(__file__), "export"))
        file = os.path.join(base_path, "policy", self.policy_name + '.json')
        with open(file) as f:
            policy = json.load(f)
        assert policy["policy"]["phases"]["hot"]["actions"]["rollover"]["max_size"] == "50gb", policy
        assert policy["policy"]["phases"]["hot"]["actions"]["rollover"]["max_age"] == "30d", policy

        os.remove(file)
