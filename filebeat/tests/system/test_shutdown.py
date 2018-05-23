from filebeat import BaseTest
import gzip
import os
import time

"""
Tests that Filebeat shuts down cleanly.
"""


class Test(BaseTest):

    def test_shutdown(self):
        """
        Test starting and stopping Filebeat under load.
        """

        self.nasa_logs()

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            ignore_older="1h"
        )
        for i in range(1, 5):
            proc = self.start_beat(logging_args=["-e", "-v"])
            time.sleep(.5)
            proc.check_kill_and_wait()

    def test_shutdown_wait_ok(self):
        """
        Test stopping filebeat under load: wait for all events being published.
        """

        self.nasa_logs()

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            ignore_older="1h",
            shutdown_timeout="10s",
        )
        filebeat = self.start_beat()

        # Wait until first flush
        self.wait_until(
            lambda: self.log_contains_count("Publish event") > 200,
            max_timeout=15)

        filebeat.check_kill_and_wait()

        self.get_log()
        self.wait_log_contains(
            "Shutdown output timer started.",
            max_timeout=15)

        self.wait_log_contains(
            "Continue shutdown: All enqueued events being published.",
            max_timeout=15)

        # validate registry entry offset matches last published event
        registry = self.get_registry()
        outputs = self.read_output()
        offset = registry[0]["offset"]
        assert len(registry) == 1

        # we allow for a potential race in the harvester shutdown here.
        # In some cases the registry offset might match the penultimate offset.
        assert (offset == outputs[-1]["offset"] or
                offset == outputs[-2]["offset"])

    def test_shutdown_wait_timeout(self):
        """
        Test stopping filebeat under load: allow early shutdown.
        """

        self.nasa_logs()

        self.render_config_template(
            logstash={"host": "does.not.exist:12345"},
            path=os.path.abspath(self.working_dir) + "/log/*",
            ignore_older="1h",
            shutdown_timeout="1s",
        )
        filebeat = self.start_beat()

        # Wait until it tries the first time to publish
        self.wait_until(
            lambda: self.log_contains("Failed to connect"),
            max_timeout=15)

        filebeat.check_kill_and_wait()

        self.wait_until(
            lambda: self.log_contains("Shutdown output timer started."),
            max_timeout=15)

        self.wait_log_contains(
            "Continue shutdown: Time out waiting for events being published.",
            max_timeout=15)

        # check registry being really empty
        reg = self.get_registry()
        assert reg == [] or reg[0]["offset"] == 0

    def test_once(self):
        """
        Test filebeat running with the once flag.
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/test.log",
            close_eof="true",
            scan_frequency="1s"
        )

        os.mkdir(self.working_dir + "/log/")

        testfile = self.working_dir + "/log/test.log"
        file = open(testfile, 'w')

        iterations = 100
        for n in range(0, iterations):
            file.write("entry " + str(n + 1))
            file.write("\n")

        file.close()

        filebeat = self.start_beat(extra_args=["-once"])

        # Make sure all lines are read
        self.wait_until(
            lambda: self.output_has(lines=iterations),
            max_timeout=10)

        # Waits for filebeat to stop
        self.wait_until(
            lambda: self.log_contains("filebeat stopped."),
            max_timeout=15)

        # Checks that registry was written
        data = self.get_registry()
        assert len(data) == 1

    def nasa_logs(self):

        # Uncompress the nasa log file.
        nasa_log = os.path.join(self.beat_path, "tests", "files", "logs", "nasa-50k.log")
        if not os.path.isfile(nasa_log):
            with gzip.open(nasa_log + ".gz", 'rb') as infile:
                with open(nasa_log, 'w') as outfile:
                    for line in infile:
                        outfile.write(line)
        os.mkdir(self.working_dir + "/log/")
        self.copy_files(["logs/nasa-50k.log"],
                        target_dir="log")

    def test_stopping_empty_path(self):
        """
        Test filebeat stops properly when 1 input has an invalid config.
        """

        input_raw = """
- type: log
  paths: []
"""

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            input_raw=input_raw,
        )
        filebeat = self.start_beat()
        time.sleep(2)

        # Wait until first flush
        msg = "No paths were defined for input"
        self.wait_until(
            lambda: self.log_contains_count(msg) >= 1,
            max_timeout=5)

        filebeat.check_wait(exit_code=1)
