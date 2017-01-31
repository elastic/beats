import os
import metricbeat
import unittest
from nose.plugins.attrib import attr
import urllib2
import time


class ConfigTest(metricbeat.BaseTest):

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @attr('integration')
    def test_compare_config(self):
        """
        Compare full and short config output
        """

        # Copy over full and normal config

        self.copy_files(["metricbeat.yml", "metricbeat.full.yml"],
                        source_dir="../../",
                        target_dir=".")

        proc = self.start_beat(config="metricbeat.yml", output="short.log",
                               extra_args=["-E", "output.elasticsearch.hosts=['" + self.get_host() + "']"])
        time.sleep(1)
        proc.check_kill_and_wait()

        proc = self.start_beat(config="metricbeat.full.yml", output="full.log",
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

    def get_host(self):
        return 'http://' + os.getenv('ELASTICSEARCH_HOST', 'localhost') + ':' + os.getenv('ELASTICSEARCH_PORT', '9200')
