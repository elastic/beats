from base import BaseTest
from idxmgmt import IdxMgmt
import os
from nose.plugins.attrib import attr
import unittest
import shutil
import datetime

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class TestRunILM(BaseTest):

    def setUp(self):
        super(TestRunILM, self).setUp()

        # auto-derived default settings, if nothing else is set
        self.index_name = self.beat_name + "-9.9.9"
        self.alias_name = self.index_name
        self.policy_name = self.alias_name

        self.es = self.esClient()
        self.idxmgmt = IdxMgmt(self.es)
        self.idxmgmt.clean(self.beat_name)

    def renderConfig(self, **kwargs):
        self.render_config_template(
            elasticsearch={"hosts": self.get_elasticsearch_url()},
            es_template_name=self.index_name,
            **kwargs
        )

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_ilm_default(self):
        """
        Test default settings: load ilm policy, write alias and ilm template
        """
        self.renderConfig()

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
        Test respect config setting for loading ilm
        """

        self.renderConfig(ilm={"enabled": False})

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
        Test set ilm policy name
        """

        policy_name = self.beat_name + "_foo"
        self.renderConfig(ilm={"enabled": True, "policy_name": policy_name})

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
        Test ilm rollover alias setting
        """

        alias_name = self.beat_name + "_foo"
        self.renderConfig(ilm={"enabled": True, "rollover_alias": alias_name})

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains("ILM policy successfully loaded"))
        self.wait_until(lambda: self.log_contains("PublishEvents: 1 events have been published"))
        proc.check_kill_and_wait()

        self.idxmgmt.assert_ilm_template_loaded(alias_name, self.policy_name, alias_name)
        self.idxmgmt.assert_docs_written_to_alias(alias_name)
        self.idxmgmt.assert_alias_created(alias_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_pattern(self):
        """
        Test ilm pattern setting
        """

        pattern = "1"
        self.renderConfig(ilm={"enabled": True, "pattern": pattern})

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
        Test ilm pattern with date inside
        """

        pattern = "'{now/d}'"
        self.renderConfig(ilm={"enabled": True, "pattern": pattern})

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
    Copies behavior from `setup --ilm
    """

    def setUp(self):
        super(TestCommandSetupILMPolicy, self).setUp()

        self.cmd = "ilm-policy"
        # auto-derived default settings, if nothing else is set
        self.index_name = self.beat_name + "-9.9.9"
        self.alias_name = self.index_name
        self.policy_name = self.alias_name

        self.es = self.esClient()
        self.idxmgmt = IdxMgmt(self.es)
        self.idxmgmt.clean(self.beat_name)

    def renderConfig(self, **kwargs):
        self.render_config_template(
            elasticsearch={"hosts": self.get_elasticsearch_url()},
            es_template_name=self.index_name,
            **kwargs
        )

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_setup_ilm_policy_and_template(self):
        """
        Test ilm policy and template setup
        """
        self.renderConfig()

        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.cmd, "--template"])

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
        self.renderConfig()

        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.cmd])

        assert exit_code == 0
        self.idxmgmt.assert_ilm_template_loaded(self.alias_name, self.policy_name, self.alias_name)
        self.idxmgmt.assert_docs_written_to_alias(self.alias_name)
        self.idxmgmt.assert_alias_created(self.alias_name)
        self.idxmgmt.assert_policy_created(self.policy_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_setup_ilm_disabled(self):
        """
        Test ilm policy setup when ilm disabled
        """
        self.renderConfig()

        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.cmd,
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
        policy_name = self.beat_name + "_foo"
        self.renderConfig()

        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.cmd,
                                              "-E", "setup.ilm.policy_name=" + policy_name])

        assert exit_code == 0
        self.idxmgmt.assert_ilm_template_loaded(self.alias_name, policy_name, self.alias_name)
        self.idxmgmt.assert_policy_created(policy_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_rollover_alias(self):
        """
        Test ilm policy setup when rollover_alias is configured
        """
        alias_name = self.beat_name + "_foo"
        self.renderConfig()

        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.cmd,
                                              "-E", "setup.ilm.rollover_alias=" + alias_name])

        assert exit_code == 0
        self.idxmgmt.assert_ilm_template_loaded(alias_name, self.policy_name, alias_name)
        self.idxmgmt.assert_docs_written_to_alias(alias_name)
        self.idxmgmt.assert_alias_created(alias_name)


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

        self.es = self.esClient()
        self.idxmgmt = IdxMgmt(self.es)

    def assert_log_contains_policy(self, policy):
        assert self.log_contains('ILM policy successfully loaded.')
        assert self.log_contains(policy)
        assert self.log_contains('"max_age": "30d"')
        assert self.log_contains('"max_size": "50gb"')

    def assert_log_contains_write_alias(self):
        assert self.log_contains('Write alias successfully generated.')

    def test_default(self):
        """
        Test default ilm-policy export
        """

        exit_code = self.run_beat(extra_args=["export", self.cmd],
                                  config=self.config)

        assert exit_code == 0
        self.assert_log_contains_policy(self.policy_name)
        self.assert_log_contains_write_alias()

    def test_load_disabled(self):
        """
        Test ilm-policy export when ilm disabled in config
        """

        exit_code = self.run_beat(extra_args=["export", self.cmd, "-E", "setup.ilm.enabled=false"],
                                  config=self.config)

        assert exit_code == 0
        self.assert_log_contains_policy(self.policy_name)
        self.assert_log_contains_write_alias()

    def test_changed_policy_name(self):
        """
        Test ilm-policy export when ilm disabled in config


        """
        policy_name = "foo"

        exit_code = self.run_beat(extra_args=["export", self.cmd, "-E", "setup.ilm.policy_name=" + policy_name],
                                  config=self.config)

        assert exit_code == 0
        self.assert_log_contains_policy(policy_name)
        self.assert_log_contains_write_alias()
