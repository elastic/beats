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

    def test_rotating_ignore_older_larger_write_rate(self):
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            ignoreOlder="1s",
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

    def test_rotating_ignore_older_low_write_rate(self):
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            ignoreOlder="1s",
            scan_frequency="0.1s",
        )

        os.mkdir(self.working_dir + "/log/")
        testfile = self.working_dir + "/log/test.log"

        proc = self.start_filebeat(debug_selectors=['*'])

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

        # log rotate
        os.rename(testfile, testfile + ".1")
        open(testfile, 'w').close()

        # wait for file to be closed due to ignore_older
        self.wait_until(
            lambda: self.log_contains(
              "Error: Stop harvesting as file is older then ignore_older: {};".format(os.path.abspath(testfile))),
            max_timeout=10)
        time.sleep(2)

        # write second line
        lines += 1
        with open(testfile, 'a') as file:
            file.write("Line {}\n".format(lines))

        self.wait_until(
            # allow for events to be send multiple times due to log rotation
            lambda: self.output_count(lambda x: x >= lines),
            max_timeout=5)

        proc.kill_and_wait()
