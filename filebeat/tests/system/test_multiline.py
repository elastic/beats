from filebeat import TestCase
import os
import socket
import shutil

"""
Tests for the multiline log messages
"""


class Test(TestCase):
    def test_java_elasticsearch_log(self):
        """
        Test that multi lines for java logs works.
        It checks that all lines which do not start with [ are append to the last line starting with [
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            multiline=True,
            pattern="^\[",
            negate="true",
            match="after"
        )

        os.mkdir(self.working_dir + "/log/")
        shutil.copy2("../files/logs/elasticsearch-multiline-log.log", os.path.abspath(self.working_dir) + "/log/elasticsearch-multiline-log.log")

        proc = self.start_filebeat()

        # wait for the "Skipping file" log message
        self.wait_until(
            lambda: self.output_has(lines=20),
            max_timeout=10)

        proc.kill_and_wait()

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
            pattern="\\\\$",
            match="after"
        )

        os.mkdir(self.working_dir + "/log/")
        shutil.copy2("../files/logs/multiline-c-log.log", os.path.abspath(self.working_dir) + "/log/multiline-c-log.log")

        proc = self.start_filebeat()

        # wait for the "Skipping file" log message
        self.wait_until(
            lambda: self.output_has(lines=4),
            max_timeout=10)

        proc.kill_and_wait()

        output = self.read_output()

        # Check that output file has the same number of lines as the log file
        assert 4 == len(output)

    def test_max_lines(self):
        """
        Test the maximum number of lines that is sent by multiline
        All further lines are discarded
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            multiline=True,
            pattern="^\[",
            negate="true",
            match="after",
            max_lines=3
        )

        os.mkdir(self.working_dir + "/log/")
        shutil.copy2("../files/logs/elasticsearch-multiline-log.log", os.path.abspath(self.working_dir) + "/log/elasticsearch-multiline-log.log")

        proc = self.start_filebeat()

        self.wait_until(
            lambda: self.output_has(lines=20),
            max_timeout=10)

        proc.kill_and_wait()

        output = self.read_output()

        # Checks line 3 is sent
        assert True == self.log_contains("MetaDataMappingService.java:388", "output/filebeat")

        # Checks line 4 is not sent anymore
        assert False == self.log_contains("InternalClusterService.java:388", "output/filebeat")

        # Check that output file has the same number of lines as the log file
        assert 20 == len(output)

    def test_timeout(self):
        """
        Test that data is sent after timeout
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            multiline=True,
            pattern="^\[",
            negate="true",
            match="after",
        )

        os.mkdir(self.working_dir + "/log/")

        testfile = self.working_dir + "/log/test.log"
        file = open(testfile, 'w', 0)

        file.write("[2015] hello world")
        file.write("\n")
        file.write("  First Line\n")
        file.write("  Second Line\n")

        proc = self.start_filebeat()

        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)

        # Because of the timeout the following two lines should be put together
        file.write("  This should not be third\n")
        file.write("  This should not be fourth\n")
        # This starts a new pattern
        file.write("[2016] Hello world\n")
        # This line should be appended
        file.write("  First line again\n")

        self.wait_until(
            lambda: self.output_has(lines=3),
            max_timeout=10)
        proc.kill_and_wait()

        output = self.read_output()
        assert 3 == len(output)

    def test_max_bytes(self):
        """
        Test the maximum number of bytes that is sent
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            multiline=True,
            pattern="^\[",
            negate="true",
            match="after",
            max_bytes=60
        )

        os.mkdir(self.working_dir + "/log/")
        shutil.copy2("../files/logs/elasticsearch-multiline-log.log", os.path.abspath(self.working_dir) + "/log/elasticsearch-multiline-log.log")

        proc = self.start_filebeat()

        self.wait_until(
            lambda: self.output_has(lines=20),
            max_timeout=10)

        proc.kill_and_wait()

        output = self.read_output()

        # Check that first 60 chars are sent
        assert True == self.log_contains("cluster.metadata", "output/filebeat")

        # Checks that chars aferwards are not sent
        assert False == self.log_contains("Zach", "output/filebeat")

        # Check that output file has the same number of lines as the log file
        assert 20 == len(output)
