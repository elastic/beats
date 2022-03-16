#!/usr/bin/env python3

from filebeat import BaseTest
import os
import shutil
import sys
import time
import unittest

from beat.beat import Proc

"""
Tests for the input functionality.
"""


class Test(BaseTest):
    def test_fixup_registry_entries_with_global_id(self):
        """
        Should update the registry correctly when there were entries using the
        default '.global' input ID
        """
        testfile = self.working_dir + "/log/test.log"
        self.render_config_template(
            template_name="filebeat-duplicated-id",
            path=testfile,
        )
        os.mkdir(self.working_dir + "/log/")
        file = open(testfile, 'w')
        offset = 0 # keep count of the bytes written
        for n in range(0, 5):
            offset += file.write("hello world\n")
        file.close()

        os.makedirs(self.registry.path)
        registry_file = self.registry.path + "/log.json"

        template_path = "./tests/system/input/registry-fix-global-id.j2"
        self.render_template(template_path, registry_file, log_file=testfile, offset=offset)

        shutil.copyfile("./tests/system/input/registry-meta.json", self.registry.path+"/meta.json")

        proc = self.start_beat()

        # wait for the "Start next scan" log message, this means the file has been
        # fully harvested
        self.wait_until(
            lambda: self.log_contains(
                "Start next scan"),
            max_timeout=10)

        proc.check_kill_and_wait()

        reg = self.access_registry()
        self.wait_until(reg.exists)
        entries = [entry for entry in reg.load()]

        # We got the latest entry for each key in the registry,
        # they're ordered by the time of creation, so we can be sure the
        # first uses the old, '.global' ID, and the second is the 'fixed' one
        global_entry = entries[0]
        fixed_entry = entries[1]

        # Compare the key excluding the inode and device ID bits
        assert global_entry['_key'].startswith('filestream::.global::native::'),
        "old key must contain '.global' ID"
        assert fixed_entry['_key'].startswith('filestream::test-fix-global-id::native::'),
        "key in registry does not have the expected ID"

        # Compare the TTL because it indicates if the entry has been 'removed' or not
        assert global_entry['ttl'] == 0, "ttl must be 0 because that's the effect of 'removing' the entry from the registry"
        assert fixed_entry['ttl'] != 0, "ttl must not be 0 because the entry has recently been added"

    def test_ignore_older_files(self):
        """
        Should ignore files there were not modified for longer then
        the `ignore_older` setting.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            ignore_older="1s"
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

        proc = self.start_beat()

        # wait for the "Skipping file" log message
        self.wait_until(
            lambda: self.log_contains(
                "Ignore file because ignore_older reached"),
            max_timeout=10)

        proc.check_kill_and_wait()

    def test_not_ignore_old_files(self):
        """
        Should not ignore files there were modified more recent than
        the ignore_older settings.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            ignore_older="15s"
        )

        os.mkdir(self.working_dir + "/log/")

        testfile = self.working_dir + "/log/test.log"
        file = open(testfile, 'w')
        iterations = 5
        for n in range(0, iterations):
            file.write("hello world")  # 11 chars
            file.write("\n")  # 1 char
        file.close()

        proc = self.start_beat()

        self.wait_until(
            lambda: self.output_has(lines=iterations), max_timeout=10)

        proc.check_kill_and_wait()

        objs = self.read_output()
        assert len(objs) == 5

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

        filebeat = self.start_beat()

        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=15)

        filebeat.check_kill_and_wait()

        output = self.read_output()

        # Check that output file has the same number of lines as the log file
        assert 1 == len(output)
        assert output[0]["message"] == "line in log file"

    def test_rotating_close_inactive_low_write_rate(self):
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            ignore_older="10s",
            close_inactive="1s",
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

        # log rotate
        os.rename(testfile, testfile + ".1")
        open(testfile, 'w').close()

        # wait for file to be closed due to close_inactive
        self.wait_until(
            lambda: self.log_contains("Closing file"),
            max_timeout=10)

        # wait a bit longer (on 1.0.1 this would cause the harvester
        # to get in a state that resulted in it watching the wrong
        # inode for changes)
        time.sleep(2)

        # write second line
        lines += 1
        with open(testfile, 'a') as file:
            file.write("Line {}\n".format(lines))

        self.wait_until(
            # allow for events to be send multiple times due to log rotation
            lambda: self.output_count(lambda x: x >= lines),
            max_timeout=5)

        filebeat.check_kill_and_wait()

    def test_shutdown_no_inputs(self):
        """
        In case no inputs are defined, filebeat must shut down and report an error
        """
        self.render_config_template(
            inputs=False,
        )

        filebeat = self.start_beat()

        self.wait_until(
            lambda: self.log_contains(
                "no modules or inputs enabled"),
            max_timeout=10)

        filebeat.check_wait(exit_code=1)

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

    def test_files_added_late(self):
        """
        Tests that inputs stay running even though no harvesters are started yet
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
        )

        os.mkdir(self.working_dir + "/log/")

        filebeat = self.start_beat()

        # wait until first 3 scans
        self.wait_until(
            lambda: self.log_contains_count("Start next scan") > 3,
            max_timeout=10)

        testfile = self.working_dir + "/log/test.log"
        with open(testfile, 'a') as file:
            file.write("Hello World1\n")
            file.write("Hello World2\n")

        # wait for log to be read
        self.wait_until(
            lambda: self.output_has(lines=2),
            max_timeout=15)

        filebeat.check_kill_and_wait()

    def test_close_inactive(self):
        """
        Test that close_inactive closes the file but reading
        is picked up again after scan_frequency
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            ignore_older="1h",
            close_inactive="1s",
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

        # wait for file to be closed due to close_inactive
        self.wait_until(
            lambda: self.log_contains("Closing file"),
            max_timeout=10)

        # write second line
        lines += 1
        with open(testfile, 'a') as file:
            file.write("Line {}\n".format(lines))

        self.wait_until(
            # allow for events to be sent multiple times due to log rotation
            lambda: self.output_count(lambda x: x >= lines),
            max_timeout=5)

        filebeat.check_kill_and_wait()

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

    def test_skip_symlinks(self):
        """
        Test that symlinks are skipped
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
        )

        os.mkdir(self.working_dir + "/log/")
        testfile = self.working_dir + "/log/test-2016.log"
        symlink_file = self.working_dir + "/log/test.log"

        # write first line
        with open(testfile, 'a') as file:
            file.write("Hello world\n")

        if os.name == "nt":
            import win32file
            win32file.CreateSymbolicLink(symlink_file, testfile, 0)
        else:
            os.symlink(testfile, symlink_file)

        filebeat = self.start_beat()

        # wait for file to be skipped
        self.wait_until(
            lambda: self.log_contains("skipped as it is a symlink"),
            max_timeout=10)

        # wait for log to be read
        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=15)

        filebeat.check_kill_and_wait()

        data = self.read_output()

        # Make sure there is only one entry, means it didn't follow the symlink
        assert len(data) == 1

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
            lambda: self.output_has_message("entry2", output_file="output/filebeat-"+self.today+"-1.ndjson"),
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
