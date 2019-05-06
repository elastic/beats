from base import BaseTest
from idxmgmt import IdxMgmt
import os
from nose.plugins.attrib import attr
import unittest
import shutil

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class Test(BaseTest):

    def test_index_modified(self):
        """
        Test that beat stops in case elasticsearch index is modified and pattern not
        """
        self.render_config_template(
            elasticsearch={"index": "test"},
        )

        exit_code = self.run_beat()

        assert exit_code == 1
        assert self.log_contains(
            "setup.template.name and setup.template.pattern have to be set if index name is modified") is True

    def test_index_not_modified(self):
        """
        Test that beat starts running if elasticsearch output is set
        """
        self.render_config_template(
            elasticsearch={"hosts": "localhost:9200"},
        )

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        proc.check_kill_and_wait()

    def test_index_modified_no_pattern(self):
        """
        Test that beat stops in case elasticsearch index is modified and pattern not
        """
        self.render_config_template(
            elasticsearch={"index": "test"},
            es_template_name="test",
        )

        exit_code = self.run_beat()

        assert exit_code == 1
        assert self.log_contains(
            "setup.template.name and setup.template.pattern have to be set if index name is modified") is True

    def test_index_modified_no_name(self):
        """
        Test that beat stops in case elasticsearch index is modified and name not
        """
        self.render_config_template(
            elasticsearch={"index": "test"},
            es_template_pattern="test",
        )

        exit_code = self.run_beat()

        assert exit_code == 1
        assert self.log_contains(
            "setup.template.name and setup.template.pattern have to be set if index name is modified") is True

    def test_index_with_pattern_name(self):
        """
        Test that beat starts running if elasticsearch output with modified index and pattern and name are set
        """
        self.render_config_template(
            elasticsearch={"hosts": "localhost:9200"},
            es_template_name="test",
            es_template_pattern="test-*",
        )

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        proc.check_kill_and_wait()

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_json_template(self):
        """
        Test loading of json based template
        """

        self.copy_files(["template.json"])

        path = os.path.join(self.working_dir, "template.json")

        print path
        self.render_config_template(
            elasticsearch={"hosts": self.get_host()},
            template_overwrite="true",
            template_json_enabled="true",
            template_json_path=path,
            template_json_name="bla",
        )

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains("Loading json template from file"))
        self.wait_until(lambda: self.log_contains("template with name 'bla' loaded"))
        proc.check_kill_and_wait()

        es = self.esClient()
        result = es.transport.perform_request('GET', '/_template/bla')
        assert len(result) == 1

    def get_host(self):
        return os.getenv('ES_HOST', 'localhost') + ':' + os.getenv('ES_PORT', '9200')


class TestRunTemplate(BaseTest):

    def setUp(self):
        super(TestRunTemplate, self).setUp()
        # auto-derived default settings, if nothing else is set
        self.index_name = self.beat_name + "-9.9.9"

        self.es = self.esClient()
        self.idxmgmt = IdxMgmt(self.es)
        self.idxmgmt.clean(self.beat_name)

    def renderConfig(self, **kwargs):
        self.render_config_template(
            elasticsearch={"hosts": self.get_elasticsearch_url()},
            **kwargs
        )

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_template_default(self):
        """
        Test run cmd with default settings for template
        """
        self.renderConfig()

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains("template with name 'mockbeat-9.9.9' loaded"))
        self.wait_until(lambda: self.log_contains("PublishEvents: 1 events have been published"))
        proc.check_kill_and_wait()

        self.idxmgmt.assert_ilm_template_loaded(self.index_name, self.index_name, self.index_name)
        self.idxmgmt.assert_alias_created(self.index_name)
        self.idxmgmt.assert_docs_written_to_alias(self.index_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_template_created_on_ilm_policy_created(self):
        """
        Test run cmd overwrites template when new ilm policy is created
        """

        self.renderConfig()

        exit_code = self.run_beat(extra_args=["setup"])
        assert exit_code == 0
        self.idxmgmt.assert_ilm_template_loaded(self.index_name, self.index_name, self.index_name)

        alias_name = "foo"
        proc = self.start_beat(extra_args=["-E", "setup.template.overwrite=false",
                                           "-E", "setup.ilm.rollover_alias=" + alias_name])

        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains(alias_name))
        self.wait_until(lambda: self.log_contains("PublishEvents: 1 events have been published"))
        proc.check_kill_and_wait()

        self.idxmgmt.assert_ilm_template_loaded(alias_name, self.index_name, alias_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_template_disabled(self):
        """
        Test run cmd when loading template is disabled
        """
        self.renderConfig()

        proc = self.start_beat(extra_args=["-E", "setup.template.enabled=false"])
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains("PublishEvents: 1 events have been published"))
        proc.check_kill_and_wait()

        self.idxmgmt.assert_index_template_not_loaded(self.index_name)


class TestCommandSetupTemplate(BaseTest):
    """
    Test beat command `setup` related to template
    """

    def setUp(self):
        super(TestCommandSetupTemplate, self).setUp()

        # auto-derived default settings, if nothing else is set
        self.index_name = self.beat_name + "-9.9.9"
        self.setupCmd = "--template"

        self.es = self.esClient()
        self.idxmgmt = IdxMgmt(self.es)
        self.idxmgmt.clean(self.beat_name)

    def renderConfig(self, **kwargs):
        self.render_config_template(
            elasticsearch={"hosts": self.get_elasticsearch_url()},
            **kwargs
        )

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_setup(self):
        """
        Test setup cmd with all subcommands
        """
        self.renderConfig()

        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup"])

        assert exit_code == 0
        self.idxmgmt.assert_ilm_template_loaded(self.index_name, self.index_name, self.index_name)
        self.idxmgmt.assert_alias_created(self.index_name)
        self.idxmgmt.assert_policy_created(self.index_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_setup_template_default(self):
        """
        Test template setup with default config
        """
        self.renderConfig()

        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.setupCmd])

        assert exit_code == 0
        self.idxmgmt.assert_ilm_template_loaded(self.index_name, self.index_name, self.index_name)
        self.idxmgmt.assert_index_template_index_pattern(self.index_name, [self.index_name + "-*"])

        # when running `setup --template`
        # write_alias and rollover_policy related to ILM are also created
        self.idxmgmt.assert_alias_created(self.index_name)
        self.idxmgmt.assert_policy_created(self.index_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_setup_template_disabled(self):
        """
        Test template setup when ilm disabled
        """
        self.renderConfig()

        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.setupCmd,
                                              "-E", "setup.template.enabled=false"])

        assert exit_code == 0
        self.idxmgmt.assert_index_template_not_loaded(self.index_name)

        # when running `setup --template` and `setup.template.enabled=false`
        # write_alias and rollover_policy related to ILM are still created
        self.idxmgmt.assert_alias_created(self.index_name)
        self.idxmgmt.assert_policy_created(self.index_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_setup_template_with_opts(self):
        """
        Test template setup with config options
        """
        self.renderConfig()

        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.setupCmd,
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
    def test_setup_template_with_ilm_changed_pattern(self):
        """
        Test template setup with changed ilm.rollover_alias config
        """
        self.renderConfig()
        alias_name = self.beat_name + "_foo"

        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.setupCmd,
                                              "-E", "setup.ilm.rollover_alias=" + alias_name])

        assert exit_code == 0
        self.idxmgmt.assert_ilm_template_loaded(alias_name, self.index_name, alias_name)
        self.idxmgmt.assert_index_template_index_pattern(alias_name, [alias_name + "-*"])

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_template_created_on_ilm_policy_created(self):
        """
        Test run cmd overwrites template when new ilm policy is created
        """

        self.renderConfig()

        exit_code = self.run_beat(extra_args=["setup"])
        assert exit_code == 0
        self.idxmgmt.assert_ilm_template_loaded(self.index_name, self.index_name, self.index_name)

        alias_name = self.beat_name + "_foo"
        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", "--template",
                                              "-E", "setup.template.overwrite=false",
                                              "-E", "setup.ilm.rollover_alias=" + alias_name])
        assert exit_code == 0

        self.idxmgmt.assert_ilm_template_loaded(alias_name, self.index_name, alias_name)


class TestCommandExportTemplate(BaseTest):
    """
    Test beat command `export template`
    """

    def setUp(self):
        super(TestCommandExportTemplate, self).setUp()

        self.config = "libbeat.yml"
        self.output = os.path.join(self.working_dir, self.config)
        shutil.copy(os.path.join(self.beat_path, "fields.yml"), self.output)
        self.template_name = self.beat_name + "-9.9.9"

        self.es = self.esClient()
        self.idxmgmt = IdxMgmt(self.es)

    def assert_log_contains_template(self, template, index_pattern):
        assert self.log_contains('Loaded index template')
        assert self.log_contains(template)
        assert self.log_contains(index_pattern)

    def test_default(self):
        """
        Test export template works
        """
        self.render_config_template(self.beat_name, self.output,
                                    fields=self.output)
        exit_code = self.run_beat(
            extra_args=["export", "template"],
            config=self.config)

        assert exit_code == 0
        self.assert_log_contains_template(self.template_name, self.template_name + "-*")

    def test_changed_index_pattern(self):
        """
        Test export template with changed index pattern
        """
        self.render_config_template(self.beat_name, self.output,
                                    fields=self.output)
        alias_name = "mockbeat-ilm-index-pattern"

        exit_code = self.run_beat(
            extra_args=["export", "template",
                        "-E", "setup.ilm.rollover_alias=" + alias_name],
            config=self.config)

        assert exit_code == 0
        self.assert_log_contains_template(self.template_name, alias_name + "-*")

    def test_load_disabled(self):
        """
        Test template also exported when disabled in config
        """
        self.render_config_template(self.beat_name, self.output,
                                    fields=self.output)
        exit_code = self.run_beat(
            extra_args=["export", "template", "-E", "setup.template.enabled=false"],
            config=self.config)

        assert exit_code == 0
        self.assert_log_contains_template(self.template_name, self.template_name + "-*")
