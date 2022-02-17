import json
import logging
import os
import pytest
import shutil
import unittest

from base import BaseTest
from idxmgmt import IdxMgmt

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
            "setup.template.name and setup.template.pattern have to be set if index name is modified")

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
            "setup.template.name and setup.template.pattern have to be set if index name is modified")

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
            "setup.template.name and setup.template.pattern have to be set if index name is modified")

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
    @pytest.mark.tag('integration')
    def test_json_template(self):
        """
        Test loading of json based template
        """

        template_name = "bla"
        es = self.es_client()
        self.copy_files(["template.json"])
        path = os.path.join(self.working_dir, "template.json")
        print(path)

        self.render_config_template(
            elasticsearch={"hosts": self.get_host()},
            template_overwrite="true",
            template_json_enabled="true",
            template_json_path=path,
            template_json_name=template_name,
        )

        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains("Loading json template from file"))
        self.wait_until(lambda: self.log_contains('Template with name \\\"bla\\\" loaded.'))
        proc.check_kill_and_wait()

        result = es.transport.perform_request('GET', '/_index_template/' + template_name)
        assert len(result) == 1

    def get_host(self):
        return os.getenv('ES_HOST', 'localhost') + ':' + os.getenv('ES_PORT', '9200')


class TestRunTemplate(BaseTest):
    """
    Test run cmd with focus on template setup
    """

    def setUp(self):
        super(TestRunTemplate, self).setUp()
        # auto-derived default settings, if nothing else is set
        self.data_stream = self.beat_name + "-9.9.9"

        self.es = self.es_client()
        self.idxmgmt = IdxMgmt(self.es, self.data_stream)
        self.idxmgmt.delete(data_streams=[self.data_stream])

    def tearDown(self):
        self.idxmgmt.delete(data_streams=[self.data_stream])

    def render_config(self, **kwargs):
        self.render_config_template(
            elasticsearch={"hosts": self.get_elasticsearch_url()},
            **kwargs
        )

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_template_default(self):
        """
        Test run cmd with default settings for template
        """
        self.render_config()
        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains('Template with name \\\"mockbeat-9.9.9\\\" loaded.'))
        self.wait_until(lambda: self.log_contains("PublishEvents: 1 events have been published"))
        proc.check_kill_and_wait()

        self.idxmgmt.assert_index_template_loaded(self.data_stream)
        self.idxmgmt.assert_docs_written_to_data_stream(self.data_stream)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_template_disabled(self):
        """
        Test run cmd does not load template when disabled in config
        """
        self.render_config()
        proc = self.start_beat(extra_args=["-E", "setup.template.enabled=false"])
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        self.wait_until(lambda: self.log_contains("PublishEvents: 1 events have been published"))
        proc.check_kill_and_wait()

        self.idxmgmt.assert_index_template_not_loaded(self.data_stream)


class TestCommandSetupTemplate(BaseTest):
    """
    Test beat command `setup` with focus on template
    """

    def setUp(self):
        super(TestCommandSetupTemplate, self).setUp()

        # auto-derived default settings, if nothing else is set
        self.setupCmd = "--index-management"
        self.data_stream = self.beat_name + "-9.9.9"
        self.policy_name = self.beat_name

        self.es = self.es_client()
        self.idxmgmt = IdxMgmt(self.es, self.data_stream)
        self.idxmgmt.delete(indices=[self.data_stream], policies=[self.policy_name])
        logging.getLogger("urllib3").setLevel(logging.WARNING)
        logging.getLogger("elasticsearch").setLevel(logging.ERROR)

    def tearDown(self):
        self.idxmgmt.delete(indices=[self.data_stream], policies=[self.policy_name])

    def render_config(self, **kwargs):
        self.render_config_template(
            elasticsearch={"hosts": self.get_elasticsearch_url()},
            **kwargs
        )

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_setup(self):
        """
        Test setup cmd with template and ilm-policy subcommands
        """
        self.render_config()
        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.setupCmd])

        assert exit_code == 0
        self.idxmgmt.assert_index_template_loaded(self.data_stream)
        self.idxmgmt.assert_policy_created(self.policy_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_setup_template_default(self):
        """
        Test template setup with default config
        """
        self.render_config()
        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.setupCmd])

        assert exit_code == 0
        self.idxmgmt.assert_index_template_loaded(self.data_stream)
        self.idxmgmt.assert_index_template_index_pattern(self.data_stream, [self.data_stream])

        self.idxmgmt.assert_policy_created(self.policy_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_setup_template_disabled(self):
        """
        Test template setup when ilm disabled
        """
        self.render_config()
        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.setupCmd,
                                              "-E", "setup.template.enabled=false"])

        assert exit_code == 0
        self.idxmgmt.assert_index_template_not_loaded(self.data_stream)

        self.idxmgmt.assert_policy_created(self.policy_name)

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    @pytest.mark.tag('integration')
    def test_setup_template_with_opts(self):
        """
        Test template setup with config options
        """
        self.render_config()
        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.setupCmd,
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
    def test_template_created_on_ilm_policy_created(self):
        """
        Test template setup overwrites template when new ilm policy is created
        """

        self.render_config()
        exit_code = self.run_beat(logging_args=["-v", "-d", "*"],
                                  extra_args=["setup", self.setupCmd,
                                              "-E", "setup.ilm.enabled=false"])
        assert exit_code == 0
        self.idxmgmt.assert_index_template_loaded(self.data_stream)
        self.idxmgmt.assert_policy_not_created(self.policy_name)

        # ensure ilm policy is created, triggering overwriting existing template
        exit_code = self.run_beat(extra_args=["setup", self.setupCmd,
                                              "-E", "setup.template.overwrite=false",
                                              "-E", "setup.template.settings.index.number_of_shards=2"])
        assert exit_code == 0
        self.idxmgmt.assert_policy_created(self.policy_name)


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

    def assert_log_contains_template(self, index_pattern):
        assert self.log_contains('Loaded index template')
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
        self.assert_log_contains_template(self.template_name)

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
        self.assert_log_contains_template(self.template_name)

    def test_export_to_file_absolute_path(self):
        """
        Test export template to file with absolute file path
        """
        self.render_config_template(self.beat_name, self.output,
                                    fields=self.output)

        base_path = os.path.abspath(os.path.join(self.beat_path, os.path.dirname(__file__), "export"))
        exit_code = self.run_beat(
            extra_args=["export", "template", "--dir=" + base_path],
            config=self.config)

        assert exit_code == 0

        file = os.path.join(base_path, "template", self.template_name + '.json')
        with open(file) as f:
            template = json.load(f)
        assert 'index_patterns' in template
        assert template['index_patterns'] == [self.template_name], template

        os.remove(file)

    def test_export_to_file_relative_path(self):
        """
        Test export template to file with relative file path
        """
        self.render_config_template(self.beat_name, self.output,
                                    fields=self.output)

        path = os.path.join(os.path.dirname(__file__), "export")
        exit_code = self.run_beat(
            extra_args=["export", "template", "--dir=" + path],
            config=self.config)

        assert exit_code == 0

        base_path = os.path.abspath(os.path.join(self.beat_path, os.path.dirname(__file__), "export"))
        file = os.path.join(base_path, "template", self.template_name + '.json')
        with open(file) as f:
            template = json.load(f)
        assert 'index_patterns' in template
        assert template['index_patterns'] == [self.template_name], template

        os.remove(file)
