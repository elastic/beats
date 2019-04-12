#!/usr/bin/env python
"""Test the registrar"""

import os
import platform
import re
import shutil
import stat
import time
import unittest

from filebeat import BaseTest
from nose.plugins.skip import SkipTest


# Additional tests: to be implemented
# * Check if registrar file can be configured, set config param
# * Check "updating" of registrar file
# * Check what happens when registrar file is deleted


class Test(BaseTest):
    """Test class"""

    def test_registrar_file_content(self):
        """Check if registrar file is created correctly and content is as expected
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*"
        )
        os.mkdir(self.working_dir + "/log/")

        # Use \n as line terminator on all platforms per docs.
        line = "hello world\n"
        line_len = len(line) - 1 + len(os.linesep)
        iterations = 5
        testfile_path = self.working_dir + "/log/test.log"
        testfile = open(testfile_path, 'w')
        testfile.write(iterations * line)
        testfile.close()

        filebeat = self.start_beat()
        count = self.log_contains_count("states written")

        self.wait_until(
            lambda: self.output_has(lines=5),
            max_timeout=15)

        # Make sure states written appears one more time
        self.wait_until(
            lambda: self.log_contains("states written") > count,
            max_timeout=10)

        # wait until the registry file exist. Needed to avoid a race between
        # the logging and actual writing the file. Seems to happen on Windows.
        self.wait_until(self.has_registry, max_timeout=1)
        filebeat.check_kill_and_wait()

        # Check that a single file exists in the registry.
        data = self.get_registry()
        assert len(data) == 1

        logfile_abs_path = os.path.abspath(testfile_path)
        record = self.get_registry_entry_by_path(logfile_abs_path)

        self.assertDictContainsSubset({
            "source": logfile_abs_path,
            "offset": iterations * line_len,
        }, record)
        self.assertTrue("FileStateOS" in record)
        self.assertIsNone(record["meta"])
        file_state_os = record["FileStateOS"]

        if os.name == "nt":
            # Windows checks
            # TODO: Check for IdxHi, IdxLo, Vol in FileStateOS on Windows.
            self.assertEqual(len(file_state_os), 3)
        elif platform.system() == "SunOS":
            stat = os.stat(logfile_abs_path)
            self.assertEqual(file_state_os["inode"], stat.st_ino)

            # Python does not return the same st_dev value as Golang or the
            # command line stat tool so just check that it's present.
            self.assertTrue("device" in file_state_os)
        else:
            stat = os.stat(logfile_abs_path)
            self.assertDictContainsSubset({
                "inode": stat.st_ino,
                "device": stat.st_dev,
            }, file_state_os)

    def test_registrar_files(self):
        """
        Check that multiple files are put into registrar file
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*"
        )
        os.mkdir(self.working_dir + "/log/")

        testfile_path1 = self.working_dir + "/log/test1.log"
        testfile_path2 = self.working_dir + "/log/test2.log"
        file1 = open(testfile_path1, 'w')
        file2 = open(testfile_path2, 'w')

        iterations = 5
        for _ in range(0, iterations):
            file1.write("hello world")  # 11 chars
            file1.write("\n")  # 1 char
            file2.write("goodbye world")  # 11 chars
            file2.write("\n")  # 1 char

        file1.close()
        file2.close()

        filebeat = self.start_beat()

        self.wait_until(
            lambda: self.output_has(lines=10),
            max_timeout=15)
        # wait until the registry file exist. Needed to avoid a race between
        # the logging and actual writing the file. Seems to happen on Windows.
        self.wait_until(self.has_registry, max_timeout=1)
        filebeat.check_kill_and_wait()

        # Check that file exist
        data = self.get_registry()

        # Check that 2 files are port of the registrar file
        assert len(data) == 2

    def test_custom_registry_file_location(self):
        """
        Check that when a custom registry file is used, the path
        is created automatically.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            registry_home="a/b/c/registry",
        )
        os.mkdir(self.working_dir + "/log/")
        testfile_path = self.working_dir + "/log/test.log"
        with open(testfile_path, 'w') as testfile:
            testfile.write("hello world\n")
        filebeat = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=15)
        # wait until the registry file exist. Needed to avoid a race between
        # the logging and actual writing the file. Seems to happen on Windows.
        self.wait_until(
            lambda: self.has_registry("a/b/c/registry/filebeat/data.json"),
            max_timeout=1)
        filebeat.check_kill_and_wait()
        assert self.has_registry("a/b/c/registry/filebeat/data.json")

    def test_registry_file_default_permissions(self):
        """
        Test that filebeat default registry permission is set
        """

        if os.name == "nt":
            # This test is currently skipped on windows because file permission
            # configuration isn't implemented on Windows yet
            raise SkipTest

        registry_home = "a/b/c/registry"
        registry_file = os.path.join(registry_home, "filebeat/data.json")

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            registry_home=registry_home,
        )
        os.mkdir(self.working_dir + "/log/")
        testfile_path = self.working_dir + "/log/test.log"
        with open(testfile_path, 'w') as testfile:
            testfile.write("hello world\n")
        filebeat = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=15)
        # wait until the registry file exist. Needed to avoid a race between
        # the logging and actual writing the file. Seems to happen on Windows.
        self.wait_until(
            lambda: self.has_registry(registry_file),
            max_timeout=1)
        filebeat.check_kill_and_wait()

        self.assertEqual(self.file_permissions(registry_file), "0600")

    def test_registry_file_custom_permissions(self):
        """
        Test that filebeat registry permission is set as per configuration
        """

        if os.name == "nt":
            # This test is currently skipped on windows because file permission
            # configuration isn't implemented on Windows yet
            raise SkipTest

        registry_home = "a/b/c/registry"
        registry_file = os.path.join(registry_home, "filebeat/data.json")

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            registry_home=registry_home,
            registry_file_permissions=0644,
        )
        os.mkdir(self.working_dir + "/log/")
        testfile_path = self.working_dir + "/log/test.log"
        with open(testfile_path, 'w') as testfile:
            testfile.write("hello world\n")
        filebeat = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=15)
        # wait until the registry file exist. Needed to avoid a race between
        # the logging and actual writing the file. Seems to happen on Windows.
        self.wait_until(
            lambda: self.has_registry(registry_file),
            max_timeout=1)
        filebeat.check_kill_and_wait()

        self.assertEqual(self.file_permissions(registry_file), "0644")

    def test_registry_file_update_permissions(self):
        """
        Test that filebeat registry permission is updated along with configuration
        """

        if os.name == "nt":
            # This test is currently skipped on windows because file permission
            # configuration isn't implemented on Windows yet
            raise SkipTest

        registry_home = "a/b/c/registry_x"
        registry_file = os.path.join(registry_home, "filebeat/data.json")

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            registry_home=registry_home,
        )
        os.mkdir(self.working_dir + "/log/")
        testfile_path = self.working_dir + "/log/test.log"
        with open(testfile_path, 'w') as testfile:
            testfile.write("hello world\n")
        filebeat = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=15)
        # wait until the registry file exist. Needed to avoid a race between
        # the logging and actual writing the file. Seems to happen on Windows.
        self.wait_until(
            lambda: self.has_registry(registry_file),
            max_timeout=1)
        filebeat.check_kill_and_wait()

        self.assertEqual(self.file_permissions(registry_file), "0600")

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            registry_home="a/b/c/registry_x",
            registry_file_permissions=0644
        )

        filebeat = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=15)
        # wait until the registry file exist. Needed to avoid a race between
        # the logging and actual writing the file. Seems to happen on Windows.
        self.wait_until(
            lambda: self.has_registry(registry_file),
            max_timeout=1)

        # Wait a moment to make sure registry is completely written
        time.sleep(1)

        filebeat.check_kill_and_wait()

        self.assertEqual(self.file_permissions(registry_file), "0644")

    def test_rotating_file(self):
        """
        Checks that the registry is properly updated after a file is rotated
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            close_inactive="1s"
        )

        os.mkdir(self.working_dir + "/log/")
        testfile_path = self.working_dir + "/log/test.log"

        filebeat = self.start_beat()

        with open(testfile_path, 'w') as testfile:
            testfile.write("offset 9\n")

        self.wait_until(lambda: self.output_has(lines=1),
                        max_timeout=10)

        testfilerenamed = self.working_dir + "/log/test.1.log"
        os.rename(testfile_path, testfilerenamed)

        with open(testfile_path, 'w') as testfile:
            testfile.write("offset 10\n")

        self.wait_until(lambda: self.output_has(lines=2),
                        max_timeout=10)

        # Wait until rotation is detected
        self.wait_until(
            lambda: self.log_contains(
                "Updating state for renamed file"),
            max_timeout=10)

        time.sleep(1)

        filebeat.check_kill_and_wait()

        # Check that file exist
        data = self.get_registry()

        # Make sure the offsets are correctly set
        if os.name == "nt":
            assert self.get_registry_entry_by_path(os.path.abspath(testfile_path))["offset"] == 11
            assert self.get_registry_entry_by_path(os.path.abspath(testfilerenamed))["offset"] == 10
        else:
            assert self.get_registry_entry_by_path(os.path.abspath(testfile_path))["offset"] == 10
            assert self.get_registry_entry_by_path(os.path.abspath(testfilerenamed))["offset"] == 9

        # Check that 2 files are port of the registrar file
        assert len(data) == 2

    def test_data_path(self):
        """
        Checks that the registry file is written in a custom data path.
        """
        self.render_config_template(
            path=self.working_dir + "/test.log",
            path_data=self.working_dir + "/datapath",
            skip_registry_config=True,
        )
        with open(self.working_dir + "/test.log", "w") as testfile:
            testfile.write("test message\n")
        filebeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=1))
        filebeat.check_kill_and_wait()

        assert self.has_registry(data_path=self.working_dir+"/datapath")

    def test_rotating_file_inode(self):
        """
        Check that inodes are properly written during file rotation
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/input*",
            scan_frequency="1s",
            close_inactive="1s",
            clean_removed="false",
        )

        if os.name == "nt":
            raise SkipTest

        os.mkdir(self.working_dir + "/log/")
        testfile_path = self.working_dir + "/log/input"

        filebeat = self.start_beat()

        with open(testfile_path, 'w') as testfile:
            testfile.write("entry1\n")

        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)

        # Wait until rotation is detected
        self.wait_until(
            lambda: self.log_contains_count(
                "Registry file updated. 1 states written") >= 1,
            max_timeout=10)

        data = self.get_registry()
        assert os.stat(testfile_path).st_ino == self.get_registry_entry_by_path(
            os.path.abspath(testfile_path))["FileStateOS"]["inode"]

        testfilerenamed1 = self.working_dir + "/log/input.1"
        os.rename(testfile_path, testfilerenamed1)

        with open(testfile_path, 'w') as testfile:
            testfile.write("entry2\n")

        self.wait_until(
            lambda: self.output_has(lines=2),
            max_timeout=10)

        # Wait until rotation is detected
        self.wait_until(
            lambda: self.log_contains_count(
                "Updating state for renamed file") == 1,
            max_timeout=10)

        time.sleep(1)

        data = self.get_registry()

        assert os.stat(testfile_path).st_ino == self.get_registry_entry_by_path(
            os.path.abspath(testfile_path))["FileStateOS"]["inode"]
        assert os.stat(testfilerenamed1).st_ino == self.get_registry_entry_by_path(
            os.path.abspath(testfilerenamed1))["FileStateOS"]["inode"]

        # Rotate log file, create a new empty one and remove it afterwards
        testfilerenamed2 = self.working_dir + "/log/input.2"
        os.rename(testfilerenamed1, testfilerenamed2)
        os.rename(testfile_path, testfilerenamed1)

        with open(testfile_path, 'w') as testfile:
            testfile.write("")

        os.remove(testfilerenamed2)

        with open(testfile_path, 'w') as testfile:
            testfile.write("entry3\n")

        self.wait_until(
            lambda: self.output_has(lines=3),
            max_timeout=10)

        filebeat.check_kill_and_wait()

        data = self.get_registry()

        # Compare file inodes and the one in the registry
        assert os.stat(testfile_path).st_ino == self.get_registry_entry_by_path(
            os.path.abspath(testfile_path))["FileStateOS"]["inode"]
        assert os.stat(testfilerenamed1).st_ino == self.get_registry_entry_by_path(
            os.path.abspath(testfilerenamed1))["FileStateOS"]["inode"]

        # Check that 3 files are part of the registrar file. The deleted file
        # should never have been detected, but the rotated one should be in
        assert len(data) == 3, "Expected 3 files but got: %s" % data

    def test_restart_continue(self):
        """
        Check that file reading continues after restart
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/input*",
            scan_frequency="1s"
        )

        if os.name == "nt":
            raise SkipTest

        os.mkdir(self.working_dir + "/log/")
        testfile_path = self.working_dir + "/log/input"

        filebeat = self.start_beat()

        with open(testfile_path, 'w') as testfile:
            testfile.write("entry1\n")

        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)

        # Wait a moment to make sure registry is completely written
        time.sleep(1)

        assert os.stat(testfile_path).st_ino == self.get_registry_entry_by_path(
            os.path.abspath(testfile_path))["FileStateOS"]["inode"]

        filebeat.check_kill_and_wait()

        # Store first registry file
        registry_file = "registry/filebeat/data.json"
        shutil.copyfile(
            self.working_dir + "/" + registry_file,
            self.working_dir + "/registry.first",
        )

        # Append file
        with open(testfile_path, 'a') as testfile:
            testfile.write("entry2\n")

        filebeat = self.start_beat(output="filebeat2.log")

        # Output file was rotated
        self.wait_until(
            lambda: self.output_has(lines=1, output_file="output/filebeat.1"),
            max_timeout=10)

        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)

        filebeat.check_kill_and_wait()

        data = self.get_registry()

        # Compare file inodes and the one in the registry
        assert os.stat(testfile_path).st_ino == self.get_registry_entry_by_path(
            os.path.abspath(testfile_path))["FileStateOS"]["inode"]

        # Check that 1 files are part of the registrar file. The deleted file
        # should never have been detected
        assert len(data) == 1

        output = self.read_output()

        # Check that output file has the same number of lines as the log file
        assert len(output) == 1
        assert output[0]["message"] == "entry2"

    def test_rotating_file_with_restart(self):
        """
        Check that inodes are properly written during file rotation and restart
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/input*",
            scan_frequency="1s",
            close_inactive="1s",
            clean_removed="false"
        )

        if os.name == "nt":
            raise SkipTest

        os.mkdir(self.working_dir + "/log/")
        testfile_path = self.working_dir + "/log/input"

        filebeat = self.start_beat()

        with open(testfile_path, 'w') as testfile:
            testfile.write("entry1\n")

        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)

        # Wait a moment to make sure registry is completely written
        time.sleep(1)

        data = self.get_registry()
        assert os.stat(testfile_path).st_ino == self.get_registry_entry_by_path(
            os.path.abspath(testfile_path))["FileStateOS"]["inode"]

        testfilerenamed1 = self.working_dir + "/log/input.1"
        os.rename(testfile_path, testfilerenamed1)

        with open(testfile_path, 'w') as testfile:
            testfile.write("entry2\n")

        self.wait_until(
            lambda: self.output_has(lines=2),
            max_timeout=10)

        # Wait until rotation is detected
        self.wait_until(
            lambda: self.log_contains(
                "Updating state for renamed file"),
            max_timeout=10)

        # Wait a moment to make sure registry is completely written
        time.sleep(1)

        data = self.get_registry()

        assert os.stat(testfile_path).st_ino == self.get_registry_entry_by_path(
            os.path.abspath(testfile_path))["FileStateOS"]["inode"]
        assert os.stat(testfilerenamed1).st_ino == self.get_registry_entry_by_path(
            os.path.abspath(testfilerenamed1))["FileStateOS"]["inode"]

        filebeat.check_kill_and_wait()

        # Store first registry file
        registry_file = "registry/filebeat/data.json"
        shutil.copyfile(
            self.working_dir + "/" + registry_file,
            self.working_dir + "/registry.first",
        )

        # Rotate log file, create a new empty one and remove it afterwards
        testfilerenamed2 = self.working_dir + "/log/input.2"
        os.rename(testfilerenamed1, testfilerenamed2)
        os.rename(testfile_path, testfilerenamed1)

        with open(testfile_path, 'w') as testfile:
            testfile.write("")

        os.remove(testfilerenamed2)

        with open(testfile_path, 'w') as testfile:
            testfile.write("entry3\n")

        filebeat = self.start_beat(output="filebeat2.log")

        # Output file was rotated
        self.wait_until(
            lambda: self.output_has(lines=2, output_file="output/filebeat.1"),
            max_timeout=10)

        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)

        filebeat.check_kill_and_wait()

        data = self.get_registry()

        # Compare file inodes and the one in the registry
        assert os.stat(testfile_path).st_ino == self.get_registry_entry_by_path(
            os.path.abspath(testfile_path))["FileStateOS"]["inode"]
        assert os.stat(testfilerenamed1).st_ino == self.get_registry_entry_by_path(
            os.path.abspath(testfilerenamed1))["FileStateOS"]["inode"]

        # Check that 3 files are part of the registrar file. The deleted file
        # should never have been detected, but the rotated one should be in
        assert len(data) == 3

    def test_state_after_rotation(self):
        """
        Checks that the state is written correctly after rotation
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/input*",
            ignore_older="2m",
            scan_frequency="1s",
            close_inactive="1s"
        )

        os.mkdir(self.working_dir + "/log/")
        testfile_path1 = self.working_dir + "/log/input"
        testfile_path2 = self.working_dir + "/log/input.1"
        testfile_path3 = self.working_dir + "/log/input.2"

        with open(testfile_path1, 'w') as testfile:
            testfile.write("entry10\n")

        with open(testfile_path2, 'w') as testfile:
            testfile.write("entry0\n")

        filebeat = self.start_beat()

        self.wait_until(
            lambda: self.output_has(lines=2),
            max_timeout=10)

        # Wait a moment to make sure file exists
        time.sleep(1)
        self.get_registry()

        # Check that offsets are correct
        if os.name == "nt":
            # Under windows offset is +1 because of additional newline char
            assert self.get_registry_entry_by_path(os.path.abspath(testfile_path1))["offset"] == 9
            assert self.get_registry_entry_by_path(os.path.abspath(testfile_path2))["offset"] == 8
        else:
            assert self.get_registry_entry_by_path(os.path.abspath(testfile_path1))["offset"] == 8
            assert self.get_registry_entry_by_path(os.path.abspath(testfile_path2))["offset"] == 7

        # Rotate files and remove old one
        os.rename(testfile_path2, testfile_path3)
        os.rename(testfile_path1, testfile_path2)

        with open(testfile_path1, 'w') as testfile1:
            testfile1.write("entry200\n")

        # Remove file afterwards to make sure not inode reuse happens
        os.remove(testfile_path3)

        # Now wait until rotation is detected
        self.wait_until(
            lambda: self.log_contains(
                "Updating state for renamed file"),
            max_timeout=10)

        self.wait_until(
            lambda: self.log_contains_count(
                "Registry file updated. 2 states written.") >= 1,
            max_timeout=15)

        time.sleep(1)
        filebeat.kill_and_wait()

        # Check that offsets are correct
        if os.name == "nt":
            # Under windows offset is +1 because of additional newline char
            assert self.get_registry_entry_by_path(os.path.abspath(testfile_path1))["offset"] == 10
            assert self.get_registry_entry_by_path(os.path.abspath(testfile_path2))["offset"] == 9
        else:
            assert self.get_registry_entry_by_path(os.path.abspath(testfile_path1))["offset"] == 9
            assert self.get_registry_entry_by_path(os.path.abspath(testfile_path2))["offset"] == 8

    def test_state_after_rotation_ignore_older(self):
        """
        Checks that the state is written correctly after rotation and ignore older
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/input*",
            ignore_older="2m",
            scan_frequency="1s",
            close_inactive="1s"
        )

        os.mkdir(self.working_dir + "/log/")
        testfile_path1 = self.working_dir + "/log/input"
        testfile_path2 = self.working_dir + "/log/input.1"
        testfile_path3 = self.working_dir + "/log/input.2"

        with open(testfile_path1, 'w') as testfile1:
            testfile1.write("entry10\n")

        with open(testfile_path2, 'w') as testfile2:
            testfile2.write("entry0\n")

        # Change modification time so file extends ignore_older
        yesterday = time.time() - 3600 * 24
        os.utime(testfile_path2, (yesterday, yesterday))

        filebeat = self.start_beat()

        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)

        # Wait a moment to make sure file exists
        time.sleep(1)
        self.get_registry()

        # Check that offsets are correct
        if os.name == "nt":
            # Under windows offset is +1 because of additional newline char
            assert self.get_registry_entry_by_path(os.path.abspath(testfile_path1))["offset"] == 9
        else:
            assert self.get_registry_entry_by_path(os.path.abspath(testfile_path1))["offset"] == 8

        # Rotate files and remove old one
        os.rename(testfile_path2, testfile_path3)
        os.rename(testfile_path1, testfile_path2)

        with open(testfile_path1, 'w') as testfile1:
            testfile1.write("entry200\n")

        # Remove file afterwards to make sure not inode reuse happens
        os.remove(testfile_path3)

        # Now wait until rotation is detected
        self.wait_until(
            lambda: self.log_contains(
                "Updating state for renamed file"),
            max_timeout=10)

        self.wait_until(
            lambda: self.log_contains_count(
                "Registry file updated. 2 states written.") >= 1,
            max_timeout=15)

        # Wait a moment to make sure registry is completely written
        time.sleep(1)
        filebeat.kill_and_wait()

        # Check that offsets are correct
        if os.name == "nt":
            # Under windows offset is +1 because of additional newline char
            assert self.get_registry_entry_by_path(os.path.abspath(testfile_path1))["offset"] == 10
            assert self.get_registry_entry_by_path(os.path.abspath(testfile_path2))["offset"] == 9
        else:
            assert self.get_registry_entry_by_path(os.path.abspath(testfile_path1))["offset"] == 9
            assert self.get_registry_entry_by_path(os.path.abspath(testfile_path2))["offset"] == 8

    @unittest.skipIf(os.name == 'nt', 'flaky test https://github.com/elastic/beats/issues/8102')
    def test_clean_inactive(self):
        """
        Checks that states are properly removed after clean_inactive
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/input*",
            clean_inactive="3s",
            ignore_older="2s",
            close_inactive="0.2s",
            scan_frequency="0.1s"
        )

        file1 = "input1"
        file2 = "input2"
        file3 = "input3"

        self.input_logs.write(file1, "first file\n")
        self.input_logs.write(file2, "second file\n")

        filebeat = self.start_beat()

        self.wait_until(lambda: self.output_has(lines=2), max_timeout=10)

        # Wait until registry file is created
        self.wait_until(lambda: self.registry.exists(), max_timeout=15)
        assert self.registry.count() == 2

        # Wait until states are removed from inputs
        self.wait_until(self.logs.nextCheck("State removed for", count=2), max_timeout=15)

        # Write new file to make sure registrar is flushed again
        self.input_logs.write(file3, "third file\n")
        self.wait_until(lambda: self.output_has(lines=3), max_timeout=30)

        # Wait until state of new file is removed
        self.wait_until(self.logs.nextCheck("State removed for"), max_timeout=15)

        filebeat.check_kill_and_wait()

        # Check that the first two files were removed from the registry
        data = self.registry.load()
        assert len(data) == 1, "Expected a single file but got: %s" % data

        # Make sure the last file in the registry is the correct one and has the correct offset
        assert data[0]["offset"] == self.input_logs.size(file3)

    @unittest.skipIf(os.name == 'nt', 'flaky test https://github.com/elastic/beats/issues/7690')
    def test_clean_removed(self):
        """
        Checks that files which were removed, the state is removed
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/input*",
            scan_frequency="0.1s",
            clean_removed=True,
            close_removed=True
        )

        file1 = "input1"
        file2 = "input2"

        self.input_logs.write(file1, "file to be removed\n")
        self.input_logs.write(file2, "2\n")

        filebeat = self.start_beat()

        self.wait_until(lambda: self.output_has(lines=2), max_timeout=10)

        # Wait until registry file is created
        self.wait_until(self.registry.exists)

        # Wait until registry is updated
        self.wait_until(lambda: self.registry.count() == 2)

        self.input_logs.remove(file1)

        # Wait until states are removed from inputs
        self.wait_until(self.logs.check("Remove state for file as file removed"))

        # Add one more line to make sure registry is written
        self.input_logs.append(file2, "make sure registry is written\n")
        self.wait_until(lambda: self.output_has(lines=3), max_timeout=10)

        # Make sure all states are cleaned up
        self.wait_until(self.logs.nextCheck(re.compile("Registrar.*After: 1")))

        filebeat.check_kill_and_wait()

        # Check that the first to files were removed from the registry
        data = self.registry.load()
        assert len(data) == 1

        # Make sure the last file in the registry is the correct one and has the correct offset
        assert data[0]["offset"] == self.input_logs.size(file2)

    def test_clean_removed_with_clean_inactive(self):
        """
        Checks that files which were removed, the state is removed
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/input*",
            scan_frequency="0.1s",
            clean_removed=True,
            clean_inactive="60s",
            ignore_older="15s",
            close_removed=True
        )

        file1 = "input1"
        file2 = "input2"
        contents2 = [
            "2\n",
            "make sure registry is written\n",
        ]

        self.input_logs.write(file1, "file to be removed\n")
        self.input_logs.write(file2, contents2[0])
        filebeat = self.start_beat()

        self.wait_until(lambda: self.output_has(lines=2), max_timeout=10)

        # Wait until registry file is created
        self.wait_until(
            self.logs.nextCheck("Registry file updated. 2 states written."),
            max_timeout=15)

        count = self.registry.count()
        print("registry size: {}".format(count))
        assert count == 2

        self.input_logs.remove(file1)

        # Wait until states are removed from inputs
        self.wait_until(self.logs.nextCheck("Remove state for file as file removed"))

        # Add one more line to make sure registry is written
        self.input_logs.append(file2, contents2[1])

        self.wait_until(lambda: self.output_has(lines=3))

        # wait until next gc and until registry file has been updated
        self.wait_until(self.logs.check("Before: 1, After: 1, Pending: 1"))
        self.wait_until(self.logs.nextCheck("Registry file updated. 1 states written."))
        count = self.registry.count()
        print("registry size after remove: {}".format(count))
        assert count == 1

        filebeat.check_kill_and_wait()

        # Check that the first two files were removed from the registry
        data = self.registry.load()
        assert len(data) == 1

        # Make sure the last file in the registry is the correct one and has the correct offset
        assert data[0]["offset"] == self.input_logs.size(file2)

    def test_symlink_failure(self):
        """
        Test that filebeat does not start if a symlink is set as registry file
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
        )
        os.mkdir(self.working_dir + "/log/")

        testfile_path = self.working_dir + "/log/test.log"
        with open(testfile_path, 'w') as testfile:
            testfile.write("Hello World\n")

        registry_file = self.working_dir + "/registry/filebeat/data.json"
        link_to_file = self.working_dir + "registry.data"
        os.makedirs(self.working_dir + "/registry/filebeat")

        with open(link_to_file, 'w') as f:
            f.write("[]")

        if os.name == "nt":
            import win32file  # pylint: disable=import-error
            win32file.CreateSymbolicLink(registry_file, link_to_file, 0)
        else:
            os.symlink(link_to_file, registry_file)

        filebeat = self.start_beat()

        # Make sure states written appears one more time
        self.wait_until(
            lambda: self.log_contains("Exiting: Registry file path is not a regular file"),
            max_timeout=10)

        filebeat.check_kill_and_wait(exit_code=1)

    def test_invalid_state(self):
        """
        Test that filebeat fails starting if invalid state in registry
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
        )
        os.mkdir(self.working_dir + "/log/")
        registry_file = self.working_dir + "/registry"

        testfile_path = self.working_dir + "/log/test.log"
        with open(testfile_path, 'w') as testfile:
            testfile.write("Hello World\n")

        registry_file_path = self.working_dir + "/registry"
        with open(registry_file_path, 'w') as registry_file:
            # Write invalid state
            registry_file.write("Hello World")

        filebeat = self.start_beat()

        # Make sure states written appears one more time
        self.wait_until(
            lambda: self.log_contains(
                "Exiting: Could not start registrar: Error loading state"),
            max_timeout=10)

        filebeat.check_kill_and_wait(exit_code=1)

    def test_restart_state(self):
        """
        Make sure that states are rewritten correctly on restart and cleaned
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            close_inactive="200ms",
            ignore_older="2000ms",
        )

        init_files = ["test"+str(i)+".log" for i in range(3)]
        restart_files = ["test"+str(i+3)+".log" for i in range(1)]

        for name in init_files:
            self.input_logs.write(name, "Hello World\n")

        filebeat = self.start_beat()

        # Make sure states written appears one more time
        self.wait_until(
            self.logs.check("Ignore file because ignore_older"),
            max_timeout=10)

        filebeat.check_kill_and_wait()

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            close_inactive="200ms",
            ignore_older="2000ms",
            clean_inactive="3s",
        )

        filebeat = self.start_beat(output="filebeat2.log")
        logs = self.log_access("filebeat2.log")

        # Write additional file
        for name in restart_files:
            self.input_logs.write(name, "Hello World\n")

        # Make sure all 4 states are persisted
        self.wait_until(logs.nextCheck("input states cleaned up. Before: 4, After: 4"))

        # Wait until registry file is cleaned
        self.wait_until(logs.nextCheck("input states cleaned up. Before: 0, After: 0"))

        filebeat.check_kill_and_wait()

    def test_restart_state_reset(self):
        """
        Test that ttl is set to -1 after restart and no inputs covering it
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            clean_inactive="10s",
            ignore_older="5s"
        )
        os.mkdir(self.working_dir + "/log/")

        testfile_path = self.working_dir + "/log/test.log"

        with open(testfile_path, 'w') as testfile:
            testfile.write("Hello World\n")

        filebeat = self.start_beat()

        # Wait until state written
        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=30)

        filebeat.check_kill_and_wait()

        # Check that ttl > 0 was set because of clean_inactive
        data = self.get_registry()
        assert len(data) == 1
        assert data[0]["ttl"] > 0

        # No config file which does not match the existing state
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/test2.log",
            clean_inactive="10s",
            ignore_older="5s",
        )

        filebeat = self.start_beat(output="filebeat2.log")

        # Wait until inputs are started
        self.wait_until(
            lambda: self.log_contains_count(
                "Starting input of type: log", logfile="filebeat2.log") >= 1,
            max_timeout=10)

        filebeat.check_kill_and_wait()

        # Check that ttl was reset correctly
        data = self.get_registry()
        assert len(data) == 1
        assert data[0]["ttl"] == -2

    def test_restart_state_reset_ttl(self):
        """
        Test that ttl is reset after restart if clean_inactive changes
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/test.log",
            clean_inactive="20s",
            ignore_older="15s"
        )
        os.mkdir(self.working_dir + "/log/")

        testfile_path = self.working_dir + "/log/test.log"

        with open(testfile_path, 'w') as testfile:
            testfile.write("Hello World\n")

        filebeat = self.start_beat()

        # Wait until state written
        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=30)

        self.wait_until(
            lambda: self.log_contains("Registry file updated. 1 states written.",
                                      logfile="filebeat.log"), max_timeout=10)

        filebeat.check_kill_and_wait()

        # Check that ttl > 0 was set because of clean_inactive
        data = self.get_registry()
        assert len(data) == 1
        assert data[0]["ttl"] == 20 * 1000 * 1000 * 1000

        # New config file which does not match the existing clean_inactive
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/test.log",
            clean_inactive="40s",
            ignore_older="20s",
        )

        filebeat = self.start_beat(output="filebeat2.log")

        # Wait until new state is written

        self.wait_until(
            lambda: self.log_contains("Registry file updated",
                                      logfile="filebeat2.log"), max_timeout=10)

        filebeat.check_kill_and_wait()

        # Check that ttl was reset correctly
        data = self.get_registry()
        assert len(data) == 1
        assert data[0]["ttl"] == 40 * 1000 * 1000 * 1000

    def test_restart_state_reset_ttl_with_space(self):
        """
        Test that ttl is reset after restart if clean_inactive changes
        This time it is tested with a space in the filename to see if everything is loaded as
        expected
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/test file.log",
            clean_inactive="20s",
            ignore_older="15s"
        )
        os.mkdir(self.working_dir + "/log/")

        testfile_path = self.working_dir + "/log/test file.log"

        with open(testfile_path, 'w') as testfile:
            testfile.write("Hello World\n")

        filebeat = self.start_beat()

        # Wait until state written
        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=30)

        self.wait_until(
            lambda: self.log_contains("Registry file updated. 1 states written.",
                                      logfile="filebeat.log"), max_timeout=10)

        filebeat.check_kill_and_wait()

        # Check that ttl > 0 was set because of clean_inactive
        data = self.get_registry()
        assert len(data) == 1
        assert data[0]["ttl"] == 20 * 1000 * 1000 * 1000

        # new config file with other clean_inactive
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/test file.log",
            clean_inactive="40s",
            ignore_older="5s",
        )

        filebeat = self.start_beat(output="filebeat2.log")

        # Wait until new state is written
        self.wait_until(
            lambda: self.log_contains("Registry file updated",
                                      logfile="filebeat2.log"), max_timeout=10)

        filebeat.check_kill_and_wait()

        # Check that ttl was reset correctly
        data = self.get_registry()
        assert len(data) == 1
        assert data[0]["ttl"] == 40 * 1000 * 1000 * 1000

    def test_restart_state_reset_ttl_no_clean_inactive(self):
        """
        Test that ttl is reset after restart if clean_inactive is disabled
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/test.log",
            clean_inactive="10s",
            ignore_older="5s"
        )
        os.mkdir(self.working_dir + "/log/")

        testfile_path = self.working_dir + "/log/test.log"

        with open(testfile_path, 'w') as testfile:
            testfile.write("Hello World\n")

        filebeat = self.start_beat()

        # Wait until state written
        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=30)

        filebeat.check_kill_and_wait()

        # Check that ttl > 0 was set because of clean_inactive
        data = self.get_registry()
        assert len(data) == 1
        assert data[0]["ttl"] == 10 * 1000 * 1000 * 1000

        # New config without clean_inactive
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/test.log",
        )

        filebeat = self.start_beat(output="filebeat2.log")

        # Wait until inputs are started
        self.wait_until(
            lambda: self.log_contains("Registry file updated",
                                      logfile="filebeat2.log"), max_timeout=10)

        filebeat.check_kill_and_wait()

        # Check that ttl was reset correctly
        data = self.get_registry()
        assert len(data) == 1
        assert data[0]["ttl"] == -1

    def test_ignore_older_state(self):
        """
        Check that state is also persisted for files falling under ignore_older on startup
        without a previous state
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            close_inactive="1s",
            ignore_older="1s",
        )
        os.mkdir(self.working_dir + "/log/")

        testfile_path1 = self.working_dir + "/log/test.log"

        with open(testfile_path1, 'w') as testfile1:
            testfile1.write("Hello World\n")

        time.sleep(1)

        filebeat = self.start_beat()

        # Make sure file falls under ignore_older
        self.wait_until(
            lambda: self.log_contains("Ignore file because ignore_older reached"),
            max_timeout=10)

        # Make sure state is loaded for file
        self.wait_until(
            lambda: self.log_contains("Before: 1, After: 1"),
            max_timeout=10)

        # Make sure state is written
        self.wait_until(
            lambda: self.log_contains("Registry file updated. 1 states written."),
            max_timeout=10)

        filebeat.check_kill_and_wait()

        data = self.get_registry()
        assert len(data) == 1

        # Check that offset is set to the end of the file
        assert data[0]["offset"] == os.path.getsize(testfile_path1)

    def test_ignore_older_state_clean_inactive(self):
        """
        Check that state for ignore_older is not persisted when falling under clean_inactive
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            close_inactive="1s",
            clean_inactive="2s",
            ignore_older="1s",
        )
        os.mkdir(self.working_dir + "/log/")

        testfile_path1 = self.working_dir + "/log/test.log"

        with open(testfile_path1, 'w') as testfile1:
            testfile1.write("Hello World\n")

        time.sleep(2)

        filebeat = self.start_beat()

        # Make sure file falls under ignore_older
        self.wait_until(
            lambda: self.log_contains("Ignore file because ignore_older reached"),
            max_timeout=10)

        self.wait_until(
            lambda: self.log_contains(
                "Do not write state for ignore_older because clean_inactive reached"),
            max_timeout=10)

        # Make sure state is loaded for file
        self.wait_until(
            lambda: self.log_contains("Before: 0, After: 0"),
            max_timeout=10)

        # Make sure state is written
        self.wait_until(
            lambda: self.log_contains("Registry file updated. 0 states written."),
            max_timeout=10)

        filebeat.check_kill_and_wait()

        data = self.get_registry()
        assert len(data) == 0

    def test_registrar_files_with_input_level_processors(self):
        """
        Check that multiple files are put into registrar file with drop event processor
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            input_processors=[{
                "drop_event": {},
            }]
        )
        os.mkdir(self.working_dir + "/log/")

        testfile_path1 = self.working_dir + "/log/test1.log"
        testfile_path2 = self.working_dir + "/log/test2.log"
        file1 = open(testfile_path1, 'w')
        file2 = open(testfile_path2, 'w')

        iterations = 5
        for _ in range(0, iterations):
            file1.write("hello world")  # 11 chars
            file1.write("\n")  # 1 char
            file2.write("goodbye world")  # 11 chars
            file2.write("\n")  # 1 char

        file1.close()
        file2.close()

        filebeat = self.start_beat()

        # wait until the registry file exist. Needed to avoid a race between
        # the logging and actual writing the file. Seems to happen on Windows.
        self.wait_until(
            self.has_registry,
            max_timeout=10)

        # Wait a moment to make sure registry is completely written
        time.sleep(2)

        filebeat.check_kill_and_wait()

        # Check that file exist
        data = self.get_registry()

        # Check that 2 files are port of the registrar file
        assert len(data) == 2

        logfile_abs_path = os.path.abspath(testfile_path1)
        record = self.get_registry_entry_by_path(logfile_abs_path)

        self.assertDictContainsSubset({
            "source": logfile_abs_path,
            "offset": iterations * (len("hello world") + len(os.linesep)),
        }, record)
        self.assertTrue("FileStateOS" in record)
        file_state_os = record["FileStateOS"]

        if os.name == "nt":
            # Windows checks
            # TODO: Check for IdxHi, IdxLo, Vol in FileStateOS on Windows.
            self.assertEqual(len(file_state_os), 3)
        elif platform.system() == "SunOS":
            stat = os.stat(logfile_abs_path)
            self.assertEqual(file_state_os["inode"], stat.st_ino)

            # Python does not return the same st_dev value as Golang or the
            # command line stat tool so just check that it's present.
            self.assertTrue("device" in file_state_os)
        else:
            stat = os.stat(logfile_abs_path)
            self.assertDictContainsSubset({
                "inode": stat.st_ino,
                "device": stat.st_dev,
            }, file_state_os)

    def test_registrar_meta(self):
        """
        Check that multiple entries for the same file are on the registry when they have
        different meta
        """

        self.render_config_template(
            type='docker',
            input_raw='''
  containers:
    path: {path}
    stream: stdout
    ids:
      - container_id
- type: docker
  containers:
    path: {path}
    stream: stderr
    ids:
      - container_id
            '''.format(path=os.path.abspath(self.working_dir) + "/log/")
        )
        os.mkdir(self.working_dir + "/log/")
        os.mkdir(self.working_dir + "/log/container_id")
        testfile_path1 = self.working_dir + "/log/container_id/test.log"

        with open(testfile_path1, 'w') as f:
            for i in range(0, 10):
                f.write('{"log":"hello\\n","stream":"stdout","time":"2018-04-13T13:39:57.924216596Z"}\n')
                f.write('{"log":"hello\\n","stream":"stderr","time":"2018-04-13T13:39:57.924216596Z"}\n')

        filebeat = self.start_beat()

        self.wait_until(
            lambda: self.output_has(lines=20),
            max_timeout=15)

        # wait until the registry file exist. Needed to avoid a race between
        # the logging and actual writing the file. Seems to happen on Windows.

        self.wait_until(self.has_registry, max_timeout=1)

        filebeat.check_kill_and_wait()

        # Check registry contains 2 entries with meta
        data = self.get_registry()
        assert len(data) == 2
        assert data[0]["source"] == data[1]["source"]
        assert data[0]["meta"]["stream"] in ("stdout", "stderr")
        assert data[1]["meta"]["stream"] in ("stdout", "stderr")
        assert data[0]["meta"]["stream"] != data[1]["meta"]["stream"]
