from filebeat import BaseTest
import os
import time

"""
Tests for the multiline log messages
"""


class Test(BaseTest):

    def test_java_elasticsearch_log(self):
        """
        Test that multi lines for java logs works.
        It checks that all lines which do not start with [ are append to the last line starting with [
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            multiline=True,
            multiline_type="pattern",
            pattern=r"^\[",
            negate="true",
            match="after"
        )

        os.mkdir(self.working_dir + "/log/")
        self.copy_files(["logs/elasticsearch-multiline-log.log"],
                        target_dir="log")

        proc = self.start_beat()

        # wait for the "Skipping file" log message
        self.wait_until(
            lambda: self.output_has(lines=20),
            max_timeout=10)

        proc.check_kill_and_wait()

        output = self.read_output()

        # Check that output file has the same number of lines as the log file
        assert 20 == len(output)

    def test_c_style_log(self):
        """
        Test that multi lines for c style log works
        It checks that all lines following a line with \\ are appended to the previous line
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            multiline=True,
            multiline_type="pattern",
            pattern="\\\\$",
            match="before"
        )

        os.mkdir(self.working_dir + "/log/")
        self.copy_files(["logs/multiline-c-log.log"],
                        target_dir="log")

        proc = self.start_beat()

        # wait for the "Skipping file" log message
        self.wait_until(
            lambda: self.output_has(lines=4),
            max_timeout=10)

        proc.check_kill_and_wait()

        output = self.read_output()

        # Check that output file has the same number of lines as the log file
        assert 4 == len(output)

    def test_rabbitmq_multiline_log(self):
        """
        Test rabbitmq multiline log
        Special about this log file is that it has empty new lines
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            multiline=True,
            multiline_type="pattern",
            pattern="^=[A-Z]+",
            match="after",
            negate="true",
        )

        logentry = """=ERROR REPORT==== 3-Feb-2016::03:10:32 ===
connection <0.23893.109>, channel 3 - soft error:
{amqp_error,not_found,
            "no queue 'bucket-1' in vhost '/'",
            'queue.declare'}


"""
        os.mkdir(self.working_dir + "/log/")

        proc = self.start_beat()

        testfile = self.working_dir + "/log/rabbitmq.log"
        file = open(testfile, 'w')
        iterations = 3
        for n in range(0, iterations):
            file.write(logentry)
        file.close()

        # wait for the "Skipping file" log message
        self.wait_until(
            lambda: self.output_has(lines=3),
            max_timeout=10)

        proc.check_kill_and_wait()

        output = self.read_output()

        # Check that output file has the same number of lines as the log file
        assert 3 == len(output)

    def test_max_lines(self):
        """
        Test the maximum number of lines that is sent by multiline
        All further lines are discarded
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            multiline=True,
            multiline_type="pattern",
            pattern=r"^\[",
            negate="true",
            match="after",
            max_lines=3
        )

        os.mkdir(self.working_dir + "/log/")
        self.copy_files(["logs/elasticsearch-multiline-log.log"],
                        target_dir="log")

        proc = self.start_beat()

        self.wait_until(
            lambda: self.output_has(lines=20),
            max_timeout=10)

        proc.check_kill_and_wait()

        output = self.read_output()

        # Checks line 3 is sent
        assert True == self.log_contains(
            "MetaDataMappingService.java:388", "output/filebeat-" + self.today + ".ndjson")

        # Checks line 4 is not sent anymore
        assert False == self.log_contains(
            "InternalClusterService.java:388", "output/filebeat-" + self.today + ".ndjson")

        # Check that output file has the same number of lines as the log file
        assert 20 == len(output)

    def test_timeout(self):
        """
        Test that data is sent after timeout
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            multiline=True,
            multiline_type="pattern",
            pattern=r"^\[",
            negate="true",
            match="after",
        )

        os.mkdir(self.working_dir + "/log/")

        testfile = self.working_dir + "/log/test.log"
        file = open(testfile, 'wb', 0)

        file.write(b"[2015] hello world")
        file.write(b"\n")
        file.write(b"  First Line\n")
        file.write(b"  Second Line\n")

        proc = self.start_beat()

        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)

        # Because of the timeout the following two lines should be put together
        file.write(b"  This should not be third\n")
        file.write(b"  This should not be fourth\n")
        # This starts a new pattern
        file.write(b"[2016] Hello world\n")
        # This line should be appended
        file.write(b"  First line again\n")

        self.wait_until(
            lambda: self.output_has(lines=3),
            max_timeout=10)
        proc.check_kill_and_wait()

        output = self.read_output()
        assert 3 == len(output)

    def test_max_bytes(self):
        """
        Test the maximum number of bytes that is sent
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            multiline=True,
            multiline_type="pattern",
            pattern=r"^\[",
            negate="true",
            match="after",
            max_bytes=60
        )

        os.mkdir(self.working_dir + "/log/")
        self.copy_files(["logs/elasticsearch-multiline-log.log"],
                        target_dir="log")

        proc = self.start_beat()

        self.wait_until(
            lambda: self.output_has(lines=20),
            max_timeout=10)

        proc.check_kill_and_wait()

        output = self.read_output()

        # Check that first 60 chars are sent
        assert True == self.log_contains("cluster.metadata", "output/filebeat-" + self.today + ".ndjson")

        # Checks that chars afterwards are not sent
        assert False == self.log_contains("Zach", "output/filebeat-" + self.today + ".ndjson")

        # Check that output file has the same number of lines as the log file
        assert 20 == len(output)

    def test_close_timeout_with_multiline(self):
        """
        Test if multiline events are split up with close_timeout
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            multiline=True,
            multiline_type="pattern",
            pattern=r"^\[",
            negate="true",
            match="after",
            close_timeout="2s",
        )

        os.mkdir(self.working_dir + "/log/")

        testfile = self.working_dir + "/log/test.log"

        with open(testfile, 'wb', 0) as file:
            file.write(b"[2015] hello world")
            file.write(b"\n")
            file.write(b"  First Line\n")
            file.write(b"  Second Line\n")

        proc = self.start_beat()

        # Wait until harvester is closed because of timeout
        # This leads to the partial event above to be sent
        self.wait_until(
            lambda: self.log_contains(
                "Closing harvester because close_timeout was reached"),
            max_timeout=15)

        # Because of the timeout the following two lines should be put together
        with open(testfile, 'ab', 0) as file:
            file.write(b"  This should not be third\n")
            file.write(b"  This should not be fourth\n")
            # This starts a new pattern
            file.write(b"[2016] Hello world\n")
            # This line should be appended
            file.write(b"  First line again\n")

        self.wait_until(
            lambda: self.output_has(lines=3),
            max_timeout=10)
        proc.check_kill_and_wait()

        # close_timeout must have closed the reader exactly twice
        self.wait_until(
            lambda: self.log_contains_count(
                "Closing harvester because close_timeout was reached") >= 1,
            max_timeout=15)

        output = self.read_output()
        assert 3 == len(output)

    def test_consecutive_newline(self):
        """
        Test if consecutive multilines have an affect on multiline
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            multiline=True,
            multiline_type="pattern",
            pattern=r"^\[",
            negate="true",
            match="after",
            close_timeout="2s",
        )

        logentry1 = """[2016-09-02 19:54:23 +0000] Started 2016-09-02 19:54:23 +0000 "GET" for /gaq?path=%2FCA%2FFallbrook%2F1845-Acacia-Ln&referer=http%3A%2F%2Fwww.xxxxx.com%2FAcacia%2BLn%2BFallbrook%2BCA%2Baddresses&search_bucket=none&page_controller=v9%2Faddresses&page_action=show at 23.235.47.31
X-Forwarded-For:72.197.227.93, 23.235.47.31
Processing by GoogleAnalyticsController#index as JSON

  Parameters: {"path"=>"/CA/Fallbrook/1845-Acacia-Ln", "referer"=>"http://www.xxxx.com/Acacia+Ln+Fallbrook+CA+addresses", "search_bucket"=>"none", "page_controller"=>"v9/addresses", "page_action"=>"show"}
Completed 200 OK in 5ms (Views: 1.9ms)""".encode("utf-8")
        logentry2 = """[2016-09-02 19:54:23 +0000] Started 2016-09-02 19:54:23 +0000 "GET" for /health_check at xxx.xx.44.181
X-Forwarded-For:
SetAdCodeMiddleware.default_ad_code referer
SetAdCodeMiddleware.default_ad_code path /health_check
SetAdCodeMiddleware.default_ad_code route """.encode("utf-8")

        os.mkdir(self.working_dir + "/log/")

        testfile = self.working_dir + "/log/test.log"

        with open(testfile, 'bw', 0) as file:
            file.write(logentry1 + b"\n")
            file.write(logentry2 + b"\n")

        proc = self.start_beat()

        self.wait_until(
            lambda: self.output_has(lines=2),
            max_timeout=10)

        proc.check_kill_and_wait()

        output = self.read_output_json()
        output[0]["message"] = logentry1
        output[1]["message"] = logentry2

    def test_invalid_config(self):
        """
        Test that filebeat errors if pattern is missing config
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir + "/log/") + "*",
            multiline=True,
            multiline_type="pattern",
            match="after",
        )

        proc = self.start_beat()

        self.wait_until(lambda: self.log_contains("multiline.pattern cannot be empty") == 1)

        proc.check_kill_and_wait(exit_code=1)
