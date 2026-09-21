#!/usr/bin/env python3

from filebeat import BaseTest
import os
import sys
import time
import unittest

from beat.beat import Proc

"""
Tests for the input functionality.
"""


class Test(BaseTest):
    def test_rotating_close_inactive_larger_write_rate(self):
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            ignore_older="10s",
            close_inactive="1s",
            scan_frequency="0.1s",
        )

        os.mkdir(self.working_dir + "/log/")
        testfile = self.working_dir + "/log/test.log"

        proc = self.start_beat()
        time.sleep(1)

        rotations = 2
        iterations = 3
        for r in range(rotations):
            with open(testfile, 'wb', 0) as file:
                for n in range(iterations):
                    file.write(bytes("hello world {}\n".format(r * iterations + n), "utf-8"))
                    time.sleep(0.1)
            os.rename(testfile, testfile + str(time.time()))

        lines = rotations * iterations
        self.wait_until(
            # allow for events to be send multiple times due to log rotation
            lambda: self.output_count(lambda x: x >= lines),
            max_timeout=15)

        proc.check_kill_and_wait()

    @unittest.skip("Skipped as flaky: https://github.com/elastic/beats/issues/34982")
    def test_no_paths_defined(self):
        """
        In case a input is defined but doesn't contain any paths, input must return error which
        leads to shutdown of filebeat because of configuration error
        """
        self.render_config_template(
        )

        filebeat = self.start_beat()

        # wait for first  "Start next scan" log message
        self.wait_until(
            lambda: self.log_contains(
                "No paths were defined for "),
            max_timeout=10)

        self.wait_until(
            lambda: self.log_contains(
                "Exiting"),
            max_timeout=10)

        filebeat.check_wait(exit_code=1)

    def test_close_inactive_file_removal(self):
        """
        Test that close_inactive still applies also if the file to close was removed
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            ignore_older="1h",
            close_inactive="3s",
            scan_frequency="0.1s",
        )

        os.mkdir(self.working_dir + "/log/")
        testfile = self.working_dir + "/log/test.log"

        filebeat = self.start_beat()

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

        os.remove(testfile)

        # wait for file to be closed due to close_inactive
        self.wait_until(
            lambda: self.log_contains("Closing file"),
            max_timeout=10)

        filebeat.check_kill_and_wait()

    def test_close_inactive_file_rotation_and_removal(self):
        """
        Test that close_inactive still applies also if the file to close was removed
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/test.log",
            ignore_older="1h",
            close_inactive="3s",
            scan_frequency="0.1s",
        )

        os.mkdir(self.working_dir + "/log/")
        testfile = self.working_dir + "/log/test.log"
        renamed_file = self.working_dir + "/log/test_renamed.log"

        filebeat = self.start_beat()

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

        os.rename(testfile, renamed_file)
        os.remove(renamed_file)

        # wait for file to be closed due to close_inactive
        self.wait_until(
            lambda: self.log_contains(
                # Still checking for old file name as filename does not change in harvester
                "Closing file"),
            max_timeout=10)

        filebeat.check_kill_and_wait()

    def test_close_inactive_file_rotation_and_removal2(self):
        """
        Test that close_inactive still applies also if file was rotated,
        new file created, and rotated file removed.
        """
        log_path = os.path.abspath(os.path.join(self.working_dir, "log"))
        os.mkdir(log_path)
        testfile = os.path.join(log_path, "a.log")
        renamed_file = os.path.join(log_path, "b.log")

        self.render_config_template(
            path=testfile,
            ignore_older="1h",
            close_inactive="3s",
            scan_frequency="0.1s",
        )

        filebeat = self.start_beat()

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

        os.rename(testfile, renamed_file)

        # write second line
        lines += 1
        with open(testfile, 'a') as file:
            file.write("Line {}\n".format(lines))

        # wait for log to be read
        self.wait_until(
            lambda: self.output_has(lines=lines),
            max_timeout=15)

        os.remove(renamed_file)

        # Wait until both files are closed
        self.wait_until(
            lambda: self.log_contains_count(
                # Checking if two files were closed
                "Closing file") == 2,
            max_timeout=10)

        filebeat.check_kill_and_wait()

    def test_harvester_limit(self):
        """
        Test if harvester_limit applies
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            harvester_limit=1,
            close_inactive="1s",
            scan_frequency="1s",
        )

        os.mkdir(self.working_dir + "/log/")
        testfile1 = self.working_dir + "/log/test1.log"
        testfile2 = self.working_dir + "/log/test2.log"
        testfile3 = self.working_dir + "/log/test3.log"

        with open(testfile1, 'w') as file:
            file.write("Line1\n")

        with open(testfile2, 'w') as file:
            file.write("Line2\n")

        with open(testfile3, 'w') as file:
            file.write("Line3\n")

        filebeat = self.start_beat()

        # check that not all harvesters were started
        self.wait_until(
            lambda: self.log_contains("harvester limit reached"))

        self.wait_until(lambda: self.output_lines() > 0)

        # Make sure not all events were written so far
        data = self.read_output()
        assert len(data) < 3

        self.wait_until(lambda: self.output_has(lines=3))

        data = self.read_output()
        assert len(data) == 3

        filebeat.check_kill_and_wait()

    def test_input_filter_dropfields(self):
        """
        Check drop_fields filtering action at a input level
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
            input_processors=[{
                "drop_fields": {
                    "fields": ["log.offset"],
                },
            }]
        )
        with open(self.working_dir + "/test.log", "w") as f:
            f.write("test message\n")

        filebeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=1))
        filebeat.check_kill_and_wait()

        output = self.read_output(
            required_fields=["@timestamp"],
        )[0]
        assert "offset" not in output
        assert "message" in output

    def test_input_filter_includefields(self):
        """
        Check include_fields filtering action at a input level
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
            input_processors=[{
                "include_fields": {
                    "fields": ["log.offset"],
                },
            }]
        )
        with open(self.working_dir + "/test.log", "w") as f:
            f.write("test message\n")

        filebeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=1))
        filebeat.check_kill_and_wait()

        output = self.read_output(
            required_fields=["@timestamp"],
        )[0]
        assert "message" not in output
        assert "log.offset" in output

    def test_restart_recursive_glob(self):
        """
        Check that file reading via recursive glob patterns continues after restart
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/**",
            scan_frequency="1s"
        )

        testfile_dir = os.path.join(self.working_dir, "log", "some", "other", "subdir")
        os.makedirs(testfile_dir)
        testfile_path = os.path.join(testfile_dir, "input")

        filebeat = self.start_beat()

        with open(testfile_path, 'w') as testfile:
            testfile.write("entry1\n")

        self.wait_until(
            lambda: self.output_has_message("entry1"),
            max_timeout=10,
            name="output contains 'entry1'")

        filebeat.check_kill_and_wait()

        # Append to file
        with open(testfile_path, 'a') as testfile:
            testfile.write("entry2\n")

        filebeat = self.start_beat()

        self.wait_until(
            lambda: self.output_has_message("entry2", output_file="output/filebeat-" + self.today + "-1.ndjson"),
            max_timeout=10,
            name="output contains 'entry2'")

        filebeat.check_kill_and_wait()

    def test_disable_recursive_glob(self):
        """
        Check that the recursive glob can be disabled from the config.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/**",
            scan_frequency="1s",
            disable_recursive_glob=True,
        )

        testfile_dir = os.path.join(self.working_dir, "log", "some", "other", "subdir")
        os.makedirs(testfile_dir)
        testfile_path = os.path.join(testfile_dir, "input")
        filebeat = self.start_beat()
        self.wait_until(
            lambda: self.log_contains(
                "recursive glob disabled"),
            max_timeout=10)
        filebeat.check_kill_and_wait()

    def test_input_processing_pipeline_disable_host(self):
        """
        Check processing_pipeline.disable_host in input config.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/test.log",
            publisher_pipeline={
                "disable_host": True,
            },
        )
        with open(self.working_dir + "/test.log", "w") as f:
            f.write("test message\n")

        filebeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=1))
        filebeat.check_kill_and_wait()

        output = self.read_output()
        assert "host.name" not in output[0]

    def test_path_based_identity_tracking(self):
        """
        Renamed files are picked up again as the path of the file has changed.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            close_eof="true",
            input_raw="  file_identity.path: ~",
        )

        testfile = os.path.join(self.working_dir, "log", "test.log")
        self.__write_hello_word_to_test_input_file(testfile)

        proc = self.start_beat()

        # wait until the file is picked up
        self.wait_until(lambda: self.output_has(lines=1))

        renamedfile = os.path.join(self.working_dir, "log", "renamed.log")
        os.rename(testfile, renamedfile)

        # wait until the both messages are received by the output
        self.wait_until(lambda: self.output_has(lines=2))
        proc.check_kill_and_wait()

        # assert that renaming of the file went undetected
        assert not self.log_contains("File rename was detected:" + testfile + " -> " + renamedfile)

    @unittest.skip("Skipped as flaky: https://github.com/elastic/beats/issues/20010")
    @unittest.skipIf(sys.platform.startswith("win"), "inode_marker is not supported on windows")
    def test_inode_marker_based_identity_tracking(self):
        """
        File is picked up again if the contents of the marker file changes.
        """

        marker_location = os.path.join(self.working_dir, "marker")
        with open(marker_location, 'w') as m:
            m.write("very-unique-string")

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            close_eof="true",
            input_raw="  file_identity.inode_marker.path: " + marker_location,
        )

        testfile = os.path.join(self.working_dir, "log", "test.log")
        self.__write_hello_word_to_test_input_file(testfile)

        proc = self.start_beat()

        # wait until the file is picked up
        self.wait_until(lambda: self.log_contains("Start harvester for new file: " + testfile))

        # change the ID in the marker file to simulate a new file
        with open(marker_location, 'w') as m:
            m.write("different-very-unique-id")

        self.wait_until(lambda: self.log_contains("Start harvester for new file: " + testfile))

        # wait until the both messages are received by the output
        self.wait_until(lambda: self.output_has(lines=2))
        proc.check_kill_and_wait()

    @unittest.skipIf(sys.platform.startswith("win"), "inode_marker is not supported on windows")
    def test_inode_marker_based_identity_tracking_to_path_based(self):
        """
        File reading can be continued after file_identity is changed.
        """

        marker_location = os.path.join(self.working_dir, "marker")
        with open(marker_location, 'w') as m:
            m.write("very-unique-string")

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            input_raw="  file_identity.inode_marker.path: " + marker_location,
        )

        testfile = os.path.join(self.working_dir, "log", "test.log")
        self.__write_hello_word_to_test_input_file(testfile)

        proc = self.start_beat()

        # wait until the file is picked up
        self.wait_until(lambda: self.log_contains("Start harvester for new file: " + testfile))

        self.wait_until(lambda: self.output_has(lines=1))
        proc.check_kill_and_wait()

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            rotateonstartup="false",
            input_raw="  file_identity.path: ~",
        )

        with open(testfile, 'w+') as f:
            f.write("hello world again\n")

        proc = self.start_beat()

        # on startup output is rotated
        self.wait_until(lambda: self.output_has(lines=1, output_file="output/filebeat-" + self.today + "-1.ndjson"))
        self.wait_until(lambda: self.output_has(lines=1))
        proc.check_kill_and_wait()

    def __write_hello_word_to_test_input_file(self, testfile):
        os.mkdir(self.working_dir + "/log/")
        with open(testfile, 'w') as f:
            f.write("hello world\n")
