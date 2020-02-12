import re
import sys
import unittest
import metricbeat
import os
import time


# Further tests:
# * Mix full config modules with reloading modules
# * Load empty file
# * Add remove module
# * enabled / disable module
# * multiple files
# * Test empty file

class Test(metricbeat.BaseTest):

    @unittest.skipUnless(re.match("(?i)win|linux|darwin|freebsd|openbsd", sys.platform), "os")
    def test_reload(self):
        """
        Test basic reload
        """
        self.render_config_template(
            reload=True,
            reload_path=self.working_dir + "/configs/*.yml",
            flush_min_events=1,
        )
        proc = self.start_beat()

        os.mkdir(self.working_dir + "/configs/")

        systemConfig = """
- module: system
  metricsets: ["cpu"]
  period: 1s
"""

        with open(self.working_dir + "/configs/system.yml", 'w') as f:
            f.write(systemConfig)

        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()

    @unittest.skipUnless(re.match("(?i)win|linux|darwin|freebsd|openbsd", sys.platform), "os")
    def test_start_stop(self):
        def reload_line(
            num_runners): return "Starting reload procedure, current runners: %d" % num_runners
        """
        Test if module is properly started and stopped
        """
        self.render_config_template(
            reload=True,
            reload_path=self.working_dir + "/configs/*.yml",
            flush_min_events=1,
        )
        os.mkdir(self.working_dir + "/configs/")

        config_path = self.working_dir + "/configs/system.yml"
        proc = self.start_beat()

        # Ensure no modules are loaded
        self.wait_until(
            lambda: self.log_contains(reload_line(0)),
            max_timeout=10)

        systemConfig = """
- module: system
  metricsets: ["cpu"]
  period: 1s
"""

        with open(config_path, 'w') as f:
            f.write(systemConfig)

        # Ensure the module was successfully loaded
        self.wait_until(
            lambda: self.log_contains(reload_line(1)),
            max_timeout=10)

        # Remove config again
        os.remove(config_path)

        # Ensure the module was successfully unloaded
        self.wait_until(
            lambda: self.log_contains(reload_line(0)),
            max_timeout=10)

        time.sleep(1)

        proc.check_kill_and_wait()

    @unittest.skipUnless(re.match("(?i)win|linux|darwin|freebsd|openbsd", sys.platform), "os")
    def test_wrong_module_no_reload(self):
        """
        Test beat errors when reload is disabled and some module config is wrong
        """
        self.render_config_template(
            reload=False,
            reload_path=self.working_dir + "/configs/*.yml",
        )
        os.mkdir(self.working_dir + "/configs/")

        config_path = self.working_dir + "/configs/system.yml"
        systemConfig = """
- module: system
  metricsets: ["wrong_metricset"]
  period: 1s
"""
        with open(config_path, 'w') as f:
            f.write(systemConfig)

        exit_code = self.run_beat()

        # Wait until offset for new line is updated
        self.wait_until(
            lambda: self.log_contains("metricset 'system/wrong_metricset' not found"),
            max_timeout=10)

        assert exit_code == 1
