import re
import unittest
import os
import shutil
import time

from filebeat import BaseTest
from beat.beat import INTEGRATION_TESTS
from elasticsearch import Elasticsearch


moduleConfigTemplate = """
- module: test
  test:
    enabled: true
    var.paths:
      - {}
    input:
      scan_frequency: 1s
  auth:
    enabled: false
"""


class Test(BaseTest):

    def setUp(self):
        super(BaseTest, self).setUp()
        if INTEGRATION_TESTS:
            self.es = Elasticsearch([self.get_elasticsearch_url()])

        # Copy system module
        shutil.copytree(os.path.join(self.beat_path, "tests", "system", "module", "test"),
                        os.path.join(self.working_dir, "module", "test"))

    def test_reload(self):
        """
        Test modules basic reload
        """
        self.render_config_template(
            reload=True,
            reload_path=self.working_dir + "/configs/*.yml",
            reload_type="modules",
            inputs=False,
        )

        proc = self.start_beat()

        os.mkdir(self.working_dir + "/logs/")
        logfile = self.working_dir + "/logs/test.log"
        os.mkdir(self.working_dir + "/configs/")

        with open(self.working_dir + "/configs/system.yml.test", 'w') as f:
            f.write(moduleConfigTemplate.format(self.working_dir + "/logs/*"))
        os.rename(self.working_dir + "/configs/system.yml.test",
                  self.working_dir + "/configs/system.yml")

        with open(logfile, 'w') as f:
            f.write("Hello world\n")

        self.wait_until(lambda: self.output_lines() > 0)
        assert self.output_has_message("Hello world")
        proc.check_kill_and_wait()

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    def test_reload_writes_pipeline(self):
        """
        Test modules reload brings pipelines
        """
        self.render_config_template(
            reload=True,
            reload_path=self.working_dir + "/configs/*.yml",
            reload_type="modules",
            inputs=False,
            elasticsearch={"host": self.get_elasticsearch_url()}
        )

        proc = self.start_beat()

        os.mkdir(self.working_dir + "/logs/")
        logfile = self.working_dir + "/logs/test.log"
        os.mkdir(self.working_dir + "/configs/")

        with open(self.working_dir + "/configs/system.yml.test", 'w') as f:
            f.write(moduleConfigTemplate.format(self.working_dir + "/logs/*"))
        os.rename(self.working_dir + "/configs/system.yml.test",
                  self.working_dir + "/configs/system.yml")

        with open(logfile, 'w') as f:
            f.write("Hello world\n")

        # Check pipeline is present
        self.wait_until(lambda: any(re.match("filebeat-.*-test-test-default", key)
                                    for key in self.es.transport.perform_request("GET", "/_ingest/pipeline/").keys()))
        proc.check_kill_and_wait()

    def test_no_es_connection(self):
        """
        Test pipeline loading failures don't crash filebeat
        """
        self.render_config_template(
            reload=True,
            reload_path=self.working_dir + "/configs/*.yml",
            reload_type="modules",
            inputs=False,
            elasticsearch={"host": 'errorhost:9201'}
        )

        proc = self.start_beat()

        os.mkdir(self.working_dir + "/configs/")
        with open(self.working_dir + "/configs/system.yml.test", 'w') as f:
            f.write(moduleConfigTemplate.format(self.working_dir + "/logs/*"))
        os.rename(self.working_dir + "/configs/system.yml.test",
                  self.working_dir + "/configs/system.yml")

        self.wait_until(lambda: self.log_contains("Error loading pipeline: Error creating Elasticsearch client"))
        proc.check_kill_and_wait(0)

    def test_start_stop(self):
        """
        Test basic modules start and stop
        """
        self.render_config_template(
            reload=True,
            reload_path=self.working_dir + "/configs/*.yml",
            reload_type="modules",
            inputs=False,
        )

        proc = self.start_beat()

        os.mkdir(self.working_dir + "/logs/")
        logfile = self.working_dir + "/logs/test.log"
        os.mkdir(self.working_dir + "/configs/")

        with open(self.working_dir + "/configs/system.yml.test", 'w') as f:
            f.write(moduleConfigTemplate.format(self.working_dir + "/logs/*"))
        os.rename(self.working_dir + "/configs/system.yml.test",
                  self.working_dir + "/configs/system.yml")

        with open(logfile, 'w') as f:
            f.write("Hello world\n")

        self.wait_until(lambda: self.output_lines() == 1, max_timeout=10)
        print(self.output_lines())

        # Remove input
        with open(self.working_dir + "/configs/system.yml", 'w') as f:
            f.write("")

        # Wait until input is stopped
        self.wait_until(
            lambda: self.log_contains("Stopping runner:"),
            max_timeout=15)

        with open(logfile, 'a') as f:
            f.write("Hello world\n")

        # Wait to give a change to pick up the new line (it shouldn't)
        time.sleep(1)

        self.wait_until(lambda: self.output_lines() == 1, max_timeout=5)
        proc.check_kill_and_wait()

    def test_load_configs(self):
        """
        Test loading separate module configs
        """
        self.render_config_template(
            reload_path=self.working_dir + "/configs/*.yml",
            reload_type="modules",
            inputs=False,
        )

        os.mkdir(self.working_dir + "/logs/")
        os.mkdir(self.working_dir + "/configs/")
        logfile1 = self.working_dir + "/logs/test1.log"
        logfile2 = self.working_dir + "/logs/test2.log"

        with open(self.working_dir + "/configs/module1.yml", 'w') as f:
            f.write(moduleConfigTemplate.format(
                self.working_dir + "/logs/test1.log"))

        with open(self.working_dir + "/configs/module2.yml", 'w') as f:
            f.write(moduleConfigTemplate.format(
                self.working_dir + "/logs/test2.log"))

        proc = self.start_beat()

        with open(logfile1, 'w') as f:
            f.write("Hello 1\n")

        self.wait_until(lambda: self.output_lines() == 1)

        with open(logfile2, 'w') as f:
            f.write("Hello 2\n")

        self.wait_until(lambda: self.output_lines() == 2)

        output = self.read_output()

        # Reloading stopped.
        self.wait_until(
            lambda: self.log_contains("Loading of config files completed."),
            max_timeout=15)

        # Make sure the correct lines were picked up
        assert self.output_lines() == 2
        assert output[0]["message"] == "Hello 1"
        assert output[1]["message"] == "Hello 2"
        proc.check_kill_and_wait()

    def test_wrong_module_no_reload(self):
        """
        Test beat errors when reload is disabled and some module config is wrong
        """
        self.render_config_template(
            reload=False,
            reload_path=self.working_dir + "/configs/*.yml",
            inputs=False,
        )
        os.mkdir(self.working_dir + "/configs/")

        config_path = self.working_dir + "/configs/wrong_module.yml"
        moduleConfig = """
- module: test
  test:
    enabled: true
    wrong_field: error
    input:
      scan_frequency: 1s
"""
        with open(config_path, 'w') as f:
            f.write(moduleConfig)

        exit_code = self.run_beat()

        # Wait until offset for new line is updated
        self.wait_until(
            lambda: self.log_contains("No paths were defined for input accessing"),
            max_timeout=10)

        assert exit_code == 1
