import os
import metricbeat
import unittest
from nose.plugins.attrib import attr
import urllib2
import time


class ConfigTest(metricbeat.BaseTest):

    @unittest.skip("This is test is currently skipped as it is flaky every time log messages change.")
    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_compare_config(self):
        """
        Compare full and short config output
        """

        # Copy over full and normal config

        self.copy_files(["metricbeat.yml", "metricbeat.reference.yml"],
                        source_dir="../../",
                        target_dir=".")

        proc = self.start_beat(config="metricbeat.yml", output="short.log",
                               extra_args=["-E", "output.elasticsearch.hosts=['" + self.get_host() + "']"])
        time.sleep(1)
        proc.check_kill_and_wait()

        proc = self.start_beat(config="metricbeat.reference.yml", output="full.log",
                               extra_args=["-E", "output.elasticsearch.hosts=['" + self.get_host() + "']"])
        time.sleep(1)
        proc.check_kill_and_wait()

        # Fetch first 27 lines
        # Remove timestamp

        shortLog = []
        with open(os.path.join(self.working_dir, "short.log"), "r") as f:
            for i in range(27):
                # Remove 27 chars of timestamp
                shortLog.append(f.next()[27:])

        fullLog = []
        with open(os.path.join(self.working_dir, "full.log"), "r") as f:
            for i in range(27):
                # Remove 27 chars of timestamp
                fullLog.append(f.next()[27:])

        same = True

        for i in range(27):
            shortLine = shortLog[i]
            fullLine = fullLog[i]

            if shortLine not in fullLog:
                print(shortLine)
                print(fullLine)
                same = False

            if fullLine not in shortLog:
                print(shortLine)
                print(fullLine)
                same = False

        assert same == True

    def test_invalid_config_with_removed_settings(self):
        """
        Checks if metricbeat fails to load a module if remove settings have been used:
        """
        self.render_config_template(modules=[{
            "name": "system",
            "metricsets": ["cpu"],
            "period": "5s",
        }])

        exit_code = self.run_beat(extra_args=[
            "-E",
            "metricbeat.modules.0.filters.0.include_fields='field1,field2'"
        ])
        assert exit_code == 1
        assert self.log_contains("setting 'metricbeat.modules.0.filters'"
                                 " has been removed")

    def get_host(self):
        return 'http://' + os.getenv('ES_HOST', 'localhost') + ':' + os.getenv('ES_PORT', '9200')
