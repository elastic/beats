from filebeat import TestCase
import os
import time

"""
Tests for the prospector functionality.
"""


class Test(TestCase):

    def test_ignore_old_files(self):
        """
        Should ignore files there were not modified for longer then
        the `ignore_older` setting.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            ignoreOlder="1s"
        )

        os.mkdir(self.working_dir + "/log/")

        testfile = self.working_dir + "/log/test.log"
        file = open(testfile, 'w')
        iterations = 5
        for n in range(0, iterations):
            file.write("hello world")  # 11 chars
            file.write("\n")  # 1 char
        file.close()

        # sleep for more than ignore older
        time.sleep(2)

        proc = self.start_filebeat()

        # wait for the "Skipping file" log message
        self.wait_until(
            lambda: self.log_contains(
                "Skipping file (older than ignore older of 1s):"),
            max_timeout=10)

        proc.kill_and_wait()

    def test_not_ignore_old_files(self):
        """
        Should not ignore files there were modified more recent than
        the ignore_older settings.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            ignoreOlder="15s"
        )

        os.mkdir(self.working_dir + "/log/")

        testfile = self.working_dir + "/log/test.log"
        file = open(testfile, 'w')
        iterations = 5
        for n in range(0, iterations):
            file.write("hello world")  # 11 chars
            file.write("\n")  # 1 char
        file.close()

        proc = self.start_filebeat()

        self.wait_until(
            lambda: self.log_contains(
                "Processing 5 events"),
            max_timeout=10)

        proc.kill_and_wait()

        objs = self.read_output()
        assert len(objs) == 5

    def test_stdin(self):
        """
        Test stdin input. Checks if reading is continued after the first read.
        """
        self.render_config_template(
            path="\"-\"",
            input_type="stdin"
        )

        proc = self.start_filebeat()

        self.wait_until(
            lambda: self.log_contains(
                "Harvester started for file: -"),
            max_timeout=10)


        iterations1 = 5
        for n in range(0, iterations1):
            os.write(proc.stdin_write, "Hello World\n")

        self.wait_until(
            lambda: self.output_has(lines=iterations1),
            max_timeout=15)


        iterations2 = 10
        for n in range(0, iterations2):
            os.write(proc.stdin_write, "Hello World\n")

        self.wait_until(
            lambda: self.output_has(lines=iterations1+iterations2),
            max_timeout=15)


        proc.kill_and_wait()

        objs = self.read_output()
        assert len(objs) == iterations1+iterations2
