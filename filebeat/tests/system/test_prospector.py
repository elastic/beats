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
                "Skipping file (older than ignore older of 1s"),
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

    def test_rotating_close_older_larger_write_rate(self):
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            ignoreOlder="10s",
            closeOlder="1s",
            scan_frequency="0.1s",
        )

        os.mkdir(self.working_dir + "/log/")
        testfile = self.working_dir + "/log/test.log"

        proc = self.start_filebeat(debug_selectors=['*'])
        time.sleep(1)

        rotations = 2
        iterations = 3
        for r in range(rotations):
            with open(testfile, 'w', 0) as file:
                for n in range(iterations):
                    file.write("hello world {}\n".format(r * iterations + n))
                    time.sleep(0.1)
            os.rename(testfile, testfile + str(time.time()))

        lines = rotations * iterations
        self.wait_until(
            # allow for events to be send multiple times due to log rotation
            lambda: self.output_count(lambda x: x >= lines),
            max_timeout=15)

        proc.kill_and_wait()

    def test_exclude_files(self):

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            exclude_files=[".gz$"]
        )
        os.mkdir(self.working_dir + "/log/")

        testfile = self.working_dir + "/log/test.gz"
        file = open(testfile, 'w')
        file.write("line in gz file\n")
        file.close()

        testfile = self.working_dir + "/log/test.log"
        file = open(testfile, 'w')
        file.write("line in log file\n")
        file.close()

        filebeat = self.start_filebeat()

        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=15)

        # TODO: Find better solution when filebeat did crawl the file
        # Idea: Special flag to filebeat so that filebeat is only doing and
        # crawl and then finishes
        filebeat.kill_and_wait()

        output = self.read_output()

        # Check that output file has the same number of lines as the log file
        assert 1 == len(output)
        assert output[0]["message"] == "line in log file"


    def test_close_older(self):
        """
        Test that close_older closes the file but reading
        is picked up again after scan_frequency
        """
        self.render_config_template(
                path=os.path.abspath(self.working_dir) + "/log/*",
                ignoreOlder="1h",
                closeOlder="1s",
                scan_frequency="0.1s",
        )

        os.mkdir(self.working_dir + "/log/")
        testfile = self.working_dir + "/log/test.log"

        filebeat = self.start_filebeat(debug_selectors=['*'])

        # wait for first  "Start next scan" log message
        self.wait_until(
                lambda: self.log_contains(
                        "Start next scan"),
                max_timeout=10)

        lines = 0

        # write first line
        lines += 1
        with open(testfile, 'a') as file:
            file.write("Line {}\n".format(lines))

        # wait for log to be read
        self.wait_until(
                lambda: self.output_has(lines=lines),
                max_timeout=15)

        # wait for file to be closed due to close_older
        self.wait_until(
                lambda: self.log_contains(
                        "Closing file: {}\n".format(os.path.abspath(testfile))),
                max_timeout=10)

        # write second line
        lines += 1
        with open(testfile, 'a') as file:
            file.write("Line {}\n".format(lines))

        self.wait_until(
                # allow for events to be send multiple times due to log rotation
                lambda: self.output_count(lambda x: x >= lines),
                max_timeout=5)

        filebeat.kill_and_wait()
