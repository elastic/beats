from filebeat import BaseTest
import gzip
import os
import time
import unittest

"""
Tests that Filebeat shuts down cleanly.
"""

class Test(BaseTest):

    def setUp(self):
        super(Test, self).setUp()

        # Uncompress the nasa log file.
        nasa_log = '../files/logs/nasa-50k.log'
        if not os.path.isfile(nasa_log):
            with gzip.open('../files/logs/nasa-50k.log.gz', 'rb') as infile:
                with open(nasa_log, 'w') as outfile:
                    for line in infile:
                        outfile.write(line)
        os.mkdir(self.working_dir + "/log/")
        self.copy_files(["logs/nasa-50k.log"],
                        source_dir="../files",
                        target_dir="log")

    def test_shutdown(self):
        """
        Test starting and stopping Filebeat under load.
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            ignore_older="1h"
        )
        for i in range(1,5):
            proc = self.start_beat(logging_args=["-e", "-v"])
            time.sleep(.5)
            proc.check_kill_and_wait()

    def test_shutdown_wait_ok(self):
        """
        Test stopping filebeat under load and wait for publisher queue to be emptied.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            ignore_older="1h",
            shutdown_timeout="10s",
        )
        proc = self.start_beat(logging_args=["-e", "-v"])
        time.sleep(.5)
        proc.check_kill_and_wait()

        log = self.get_log()
        assert "Shutdown output timer started." in log
        assert "Continue shutdown: All enqueued events being published." in log

        # validate registry entry offset matches last published event
        registry = self.get_registry()
        output = self.read_output()[-1]
        assert len(registry) == 1
        assert registry[0]["offset"] == output["offset"]

    def test_shutdown_wait_timeout(self):
        """
        Test stopping filebeat under load and wait for publisher queue to be emptied.
        """
        self.render_config_template(
            logstash={"host": "does.not.exist:12345"},
            path=os.path.abspath(self.working_dir) + "/log/*",
            ignore_older="1h",
            shutdown_timeout="1s",
        )
        proc = self.start_beat(logging_args=["-e", "-v"])
        time.sleep(.5)
        proc.check_kill_and_wait()

        log = self.get_log()
        assert "Shutdown output timer started." in log
        assert "Continue shutdown: Time out waiting for events being published." in log

        # check registry being really empty
        assert self.get_registry() == []
