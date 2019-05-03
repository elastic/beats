# coding=utf-8

from filebeat import BaseTest
import os
import codecs
import time
import base64
import io
import re
import unittest
from parameterized import parameterized

"""
Test Harvesters
"""


class Test(BaseTest):

    @unittest.skip('flaky test https://github.com/elastic/beats/issues/12037')
    def test_close_renamed(self):
        """
        Checks that a file is closed when its renamed / rotated
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/test.log",
            close_renamed="true",
            clean_removed="false",
            scan_frequency="0.1s"
        )
        os.mkdir(self.working_dir + "/log/")

        testfile1 = self.working_dir + "/log/test.log"
        testfile2 = self.working_dir + "/log/test.log.rotated"
        file = open(testfile1, 'w')

        iterations1 = 5
        for n in range(0, iterations1):
            file.write("rotation file")
            file.write("\n")

        file.close()

        filebeat = self.start_beat()

        # Let it read the file
        self.wait_until(
            lambda: self.output_has(lines=iterations1), max_timeout=10)

        os.rename(testfile1, testfile2)

        file = open(testfile1, 'w', 0)
        file.write("Hello World\n")
        file.close()

        # Wait until error shows up
        self.wait_until(
            lambda: self.log_contains(
                "Closing because close_renamed is enabled"),
            max_timeout=15)

        # Let it read the file
        self.wait_until(
            lambda: self.output_has(lines=iterations1 + 1), max_timeout=10)

        # Wait until registry file is created
        self.wait_until(
            lambda: self.log_contains_count("Registry file updated") > 1)

        # Make sure new file was picked up. As it has the same file name,
        # one entry for the new and one for the old should exist
        self.wait_until(
            lambda: len(self.get_registry()) == 2, max_timeout=10)

        filebeat.check_kill_and_wait()

        # Check registry has 2 entries after shutdown
        data = self.get_registry()
        assert len(data) == 2

    @unittest.skipIf(os.name == 'nt', 'flaky test https://github.com/elastic/beats/issues/9214')
    def test_close_removed(self):
        """
        Checks that a file is closed if removed
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/test.log",
            close_removed="true",
            clean_removed="false",
            scan_frequency="0.1s"
        )
        os.mkdir(self.working_dir + "/log/")

        testfile1 = self.working_dir + "/log/test.log"
        file = open(testfile1, 'w')

        iterations1 = 5
        for n in range(0, iterations1):
            file.write("rotation file")
            file.write("\n")

        file.close()

        filebeat = self.start_beat()

        # Let it read the file
        self.wait_until(
            lambda: self.output_has(lines=iterations1), max_timeout=10)

        os.remove(testfile1)

        # Make sure state is written
        self.wait_until(
            lambda: self.log_contains_count(
                "Write registry file") > 1,
            max_timeout=10)

        # Wait until error shows up on windows
        self.wait_until(
            lambda: self.log_contains(
                "Closing because close_removed is enabled"),
            max_timeout=10)

        filebeat.check_kill_and_wait()

        data = self.get_registry()

        # Make sure the state for the file was persisted
        assert len(data) == 1

    def test_close_eof(self):
        """
        Checks that a file is closed if eof is reached
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/test.log",
            close_eof="true",
            scan_frequency="0.1s"
        )
        os.mkdir(self.working_dir + "/log/")

        testfile1 = self.working_dir + "/log/test.log"
        file = open(testfile1, 'w')

        iterations1 = 5
        for n in range(0, iterations1):
            file.write("rotation file")
            file.write("\n")

        file.close()

        filebeat = self.start_beat()

        # Let it read the file
        self.wait_until(
            lambda: self.output_has(lines=iterations1), max_timeout=10)

        # Wait until error shows up on windows
        self.wait_until(
            lambda: self.log_contains(
                "Closing because close_eof is enabled"),
            max_timeout=15)

        filebeat.check_kill_and_wait()

        data = self.get_registry()

        # Make sure the state for the file was persisted
        assert len(data) == 1

    def test_empty_line(self):
        """
        Checks that no empty events are sent for an empty line but state is still updated
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/test.log",
        )
        os.mkdir(self.working_dir + "/log/")

        logfile = self.working_dir + "/log/test.log"

        filebeat = self.start_beat()

        with open(logfile, 'w') as f:
            f.write("Hello world\n")

        # Let it read the file
        self.wait_until(
            lambda: self.output_has(lines=1), max_timeout=10)

        with open(logfile, 'a') as f:
            f.write("\n")

        expectedOffset = 13

        if os.name == "nt":
            # Two additional newline chars
            expectedOffset += 2

        # Wait until offset for new line is updated
        self.wait_until(
            lambda: self.log_contains(
                "offset: " + str(expectedOffset)),
            max_timeout=15)

        with open(logfile, 'a') as f:
            f.write("Third line\n")

        # Make sure only 2 events are written
        self.wait_until(
            lambda: self.output_has(lines=2), max_timeout=10)

        filebeat.check_kill_and_wait()

        data = self.get_registry()

        # Make sure the state for the file was persisted
        assert len(data) == 1

    def test_empty_lines_only(self):
        """
        Checks that no empty events are sent for a file with only empty lines
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/test.log",
        )
        os.mkdir(self.working_dir + "/log/")

        logfile = self.working_dir + "/log/test.log"

        filebeat = self.start_beat()

        with open(logfile, 'w') as f:
            f.write("\n")
            f.write("\n")
            f.write("\n")

        expectedOffset = 3

        if os.name == "nt":
            # Two additional newline chars
            expectedOffset += 3

        # Wait until offset for new line is updated
        self.wait_until(
            lambda: self.log_contains(
                "offset: " + str(expectedOffset)),
            max_timeout=15)

        assert os.path.isfile(self.working_dir + "/output/filebeat") == False

        filebeat.check_kill_and_wait()

        data = self.get_registry()

        # Make sure the state for the file was persisted
        assert len(data) == 1

    def test_exceed_buffer(self):
        """
        Checks that also full line is sent if lines exceeds buffer
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/test.log",
            harvester_buffer_size=10,
        )
        os.mkdir(self.working_dir + "/log/")

        logfile = self.working_dir + "/log/test.log"

        filebeat = self.start_beat()
        message = "This exceeds the buffer"

        with open(logfile, 'w') as f:
            f.write(message + "\n")

        # wait for at least one event being written
        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)

        # Wait until state is written
        self.wait_until(
            lambda: self.log_contains(
                "Registrar state updates processed"),
            max_timeout=15)

        filebeat.check_kill_and_wait()

        data = self.get_registry()
        assert len(data) == 1

        output = self.read_output_json()
        assert message == output[0]["message"]

    def test_truncated_file_open(self):
        """
        Checks if it is correctly detected if an open file is truncated
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/test.log",
        )

        os.mkdir(self.working_dir + "/log/")
        logfile = self.working_dir + "/log/test.log"

        message = "Hello World"

        filebeat = self.start_beat()

        # Write 3 lines
        with open(logfile, 'w') as f:
            f.write(message + "\n")
            f.write(message + "\n")
            f.write(message + "\n")

        # wait for the "Skipping file" log message
        self.wait_until(
            lambda: self.output_has(lines=3),
            max_timeout=10)

        # Write 1 line -> truncation
        with open(logfile, 'w') as f:
            f.write(message + "\n")

        # wait for the "Skipping file" log message
        self.wait_until(
            lambda: self.output_has(lines=4),
            max_timeout=10)

        # Test if truncation was reported properly
        self.wait_until(
            lambda: self.log_contains(
                "File was truncated as offset"),
            max_timeout=15)
        self.wait_until(
            lambda: self.log_contains(
                "File was truncated. Begin reading file from offset 0"),
            max_timeout=15)

        filebeat.check_kill_and_wait()

    def test_truncated_file_closed(self):
        """
        Checks if it is correctly detected if a closed file is truncated
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/test.log",
            close_inactive="1s",
        )

        os.mkdir(self.working_dir + "/log/")
        logfile = self.working_dir + "/log/test.log"

        message = "Hello World"

        filebeat = self.start_beat()

        # Write 3 lines
        with open(logfile, 'w') as f:
            f.write(message + "\n")
            f.write(message + "\n")
            f.write(message + "\n")

        # wait for the "Skipping file" log message
        self.wait_until(
            lambda: self.output_has(lines=3),
            max_timeout=10)

        # Wait until harvester is closed
        self.wait_until(
            lambda: self.log_contains(
                "Stopping harvester for file"),
            max_timeout=15)

        # Write 1 line -> truncation
        with open(logfile, 'w') as f:
            f.write(message + "\n")

        # wait for the "Skipping file" log message
        self.wait_until(
            lambda: self.output_has(lines=4),
            max_timeout=10)

        # Test if truncation was reported properly
        self.wait_until(
            lambda: self.log_contains(
                "Old file was truncated. Starting from the beginning"),
            max_timeout=15)

        filebeat.check_kill_and_wait()

    def test_close_timeout(self):
        """
        Checks that a file is closed after close_timeout
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/test.log",
            close_timeout="1s",
            scan_frequency="1s"
        )
        os.mkdir(self.working_dir + "/log/")

        filebeat = self.start_beat()

        testfile1 = self.working_dir + "/log/test.log"
        file = open(testfile1, 'w')

        # Write 1000 lines with a sleep between each line to make sure it takes more then 1s to complete
        iterations1 = 1000
        for n in range(0, iterations1):
            file.write("example data")
            file.write("\n")
            # Make sure some contents are written to disk so the harvested is able to read it.
            file.flush()
            os.fsync(file)
            time.sleep(0.001)

        file.close()

        # Wait until harvester is closed because of ttl
        self.wait_until(
            lambda: self.log_contains(
                "Closing harvester because close_timeout was reached"),
            max_timeout=15)

        filebeat.check_kill_and_wait()

        data = self.get_registry()
        assert len(data) == 1

        # Check that not all but some lines were read. It can happen sometimes that filebeat finishes reading ...
        assert self.output_lines() <= 1000
        assert self.output_lines() > 0

    def test_bom_utf8(self):
        """
        Test utf8 log file with bom
        Additional test here to make sure in case generation in python is not correct
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
        )

        os.mkdir(self.working_dir + "/log/")
        self.copy_files(["logs/bom8.log"],
                        target_dir="log")

        filebeat = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=7),
            max_timeout=10)

        # Check that output does not cotain bom
        output = self.read_output_json()
        assert output[0]["message"] == "#Software: Microsoft Exchange Server"

        filebeat.check_kill_and_wait()

    @parameterized.expand([
        ("utf-8", "utf-8", codecs.BOM_UTF8),
        ("utf-16be-bom", "utf-16-be", codecs.BOM_UTF16_BE),
        ("utf-16le-bom", "utf-16-le", codecs.BOM_UTF16_LE),
    ])
    def test_boms(self, fb_encoding, py_encoding, bom):
        """
        Test bom log files if bom is removed properly
        """

        os.mkdir(self.working_dir + "/log/")
        os.mkdir(self.working_dir + "/output/")

        message = "Hello World"

        # Render config with specific encoding
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/" + fb_encoding + "*",
            encoding=fb_encoding,
            output_file_filename=fb_encoding,
        )

        logfile = self.working_dir + "/log/" + fb_encoding + "test.log"

        # Write bom to file
        with codecs.open(logfile, 'wb') as file:
            file.write(bom)

        # Write hello world to file
        with codecs.open(logfile, 'a', py_encoding) as file:
            content = message + '\n'
            file.write(content)

        filebeat = self.start_beat(output=fb_encoding + ".log")

        self.wait_until(
            lambda: self.output_has(lines=1, output_file="output/" + fb_encoding),
            max_timeout=10)

        # Verify that output does not contain bom
        output = self.read_output_json(output_file="output/" + fb_encoding)
        assert output[0]["message"] == message

        filebeat.kill_and_wait()

    def test_ignore_symlink(self):
        """
        Test that symlinks are ignored
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/symlink.log",
        )

        os.mkdir(self.working_dir + "/log/")

        logfile = self.working_dir + "/log/test.log"
        symlink = self.working_dir + "/log/symlink.log"

        if os.name == "nt":
            import win32file
            win32file.CreateSymbolicLink(symlink, logfile, 0)
        else:
            os.symlink(logfile, symlink)

        with open(logfile, 'a') as file:
            file.write("Hello World\n")

        filebeat = self.start_beat()

        # Make sure symlink is skipped
        self.wait_until(
            lambda: self.log_contains(
                "skipped as it is a symlink"),
            max_timeout=15)

        filebeat.check_kill_and_wait()

    def test_symlinks_enabled(self):
        """
        Test if symlinks are harvested
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/symlink.log",
            symlinks="true",
        )

        os.mkdir(self.working_dir + "/log/")

        logfile = self.working_dir + "/log/test.log"
        symlink = self.working_dir + "/log/symlink.log"

        if os.name == "nt":
            import win32file
            win32file.CreateSymbolicLink(symlink, logfile, 0)
        else:
            os.symlink(logfile, symlink)

        with open(logfile, 'a') as file:
            file.write("Hello World\n")

        filebeat = self.start_beat()

        # Make sure content in symlink file is read
        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)

        filebeat.check_kill_and_wait()

    def test_symlink_rotated(self):
        """
        Test what happens if symlink removed and points to a new file
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/symlink.log",
            symlinks="true",
            close_removed="false",
            clean_removed="false",
        )

        os.mkdir(self.working_dir + "/log/")

        logfile1 = self.working_dir + "/log/test1.log"
        logfile2 = self.working_dir + "/log/test2.log"
        symlink = self.working_dir + "/log/symlink.log"

        if os.name == "nt":
            import win32file
            win32file.CreateSymbolicLink(symlink, logfile1, 0)
        else:
            os.symlink(logfile1, symlink)

        with open(logfile1, 'a') as file:
            file.write("Hello World1\n")

        with open(logfile2, 'a') as file:
            file.write("Hello World2\n")

        filebeat = self.start_beat()

        # Make sure state is written
        self.wait_until(
            lambda: self.log_contains_count(
                "Write registry file") > 1,
            max_timeout=10)

        # Make sure symlink is skipped
        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)

        os.remove(symlink)

        if os.name == "nt":
            import win32file
            win32file.CreateSymbolicLink(symlink, logfile2, 0)
        else:
            os.symlink(logfile2, symlink)

        with open(logfile1, 'a') as file:
            file.write("Hello World3\n")
            file.write("Hello World4\n")

        # Make sure new file and addition to old file were read
        self.wait_until(
            lambda: self.output_has(lines=4),
            max_timeout=10)

        filebeat.check_kill_and_wait()

        # Check if two different files are in registry
        data = self.get_registry()
        assert len(data) == 2, "expected to see 2 entries, got '%s'" % data

    def test_symlink_removed(self):
        """
        Tests that if a symlink to a file is removed, further data is read which is added to the original file
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/symlink.log",
            symlinks="true",
            clean_removed="false",
            close_removed="false",
        )

        os.mkdir(self.working_dir + "/log/")

        logfile = self.working_dir + "/log/test.log"
        symlink = self.working_dir + "/log/symlink.log"

        if os.name == "nt":
            import win32file
            win32file.CreateSymbolicLink(symlink, logfile, 0)
        else:
            os.symlink(logfile, symlink)

        with open(logfile, 'a') as file:
            file.write("Hello World1\n")

        filebeat = self.start_beat()

        # Make sure symlink is skipped
        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)

        os.remove(symlink)

        with open(logfile, 'a') as file:
            file.write("Hello World2\n")

        # Sleep 1s to make sure new events are not picked up
        time.sleep(1)

        # Make sure also new file was read
        self.wait_until(
            lambda: self.output_has(lines=2),
            max_timeout=10)

        filebeat.check_kill_and_wait()

        # Check if two different files are in registry
        data = self.get_registry()
        assert len(data) == 1

    def test_symlink_and_file(self):
        """
        Tests that if symlink and original file are read, that only events from one are added
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            symlinks="true",
        )

        os.mkdir(self.working_dir + "/log/")

        logfile = self.working_dir + "/log/test.log"
        symlink = self.working_dir + "/log/symlink.log"

        if os.name == "nt":
            import win32file
            win32file.CreateSymbolicLink(symlink, logfile, 0)
        else:
            os.symlink(logfile, symlink)

        with open(logfile, 'a') as file:
            file.write("Hello World1\n")

        filebeat = self.start_beat()

        # Make sure both files were read
        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)

        filebeat.check_kill_and_wait()

        # Check if two different files are in registry
        data = self.get_registry()
        assert len(data) == 1

    def test_truncate(self):
        """
        Tests what happens if file is truncated and symlink recreated
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            symlinks="true",
        )

        os.mkdir(self.working_dir + "/log/")

        logfile = self.working_dir + "/log/test.log"
        symlink = self.working_dir + "/log/symlink.log"

        if os.name == "nt":
            import win32file
            win32file.CreateSymbolicLink(symlink, logfile, 0)
        else:
            os.symlink(logfile, symlink)

        with open(logfile, 'w') as file:
            file.write("Hello World1\n")
            file.write("Hello World2\n")
            file.write("Hello World3\n")
            file.write("Hello World4\n")

        filebeat = self.start_beat()

        # Make sure both files were read
        self.wait_until(
            lambda: self.output_has(lines=4),
            max_timeout=10)

        os.remove(symlink)
        with open(logfile, 'w') as file:
            file.truncate()
            file.seek(0)

        if os.name == "nt":
            import win32file
            win32file.CreateSymbolicLink(symlink, logfile, 0)
        else:
            os.symlink(logfile, symlink)

        # Write new file with content shorter then old one
        with open(logfile, 'a') as file:
            file.write("Hello World5\n")
            file.write("Hello World6\n")
            file.write("Hello World7\n")

        # Make sure both files were read
        self.wait_until(
            lambda: self.output_has(lines=7),
            max_timeout=10)

        filebeat.check_kill_and_wait()

        # Check that only 1 registry entry as original was only truncated
        data = self.get_registry()
        assert len(data) == 1

    def test_decode_error(self):
        """
        Tests that in case of a decoding error it is handled gracefully
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            encoding="utf-16be",
        )

        os.mkdir(self.working_dir + "/log/")

        logfile = self.working_dir + "/log/test.log"

        with io.open(logfile, 'w', encoding="utf-16le") as file:
            file.write(u'hello world1')
            file.write(u"\n")
        with io.open(logfile, 'a', encoding="utf-16le") as file:
            file.write(u"\U00012345=Ra")
        with io.open(logfile, 'a', encoding="utf-16le") as file:
            file.write(u"\n")
            file.write(u"hello world2")
            file.write(u"\n")

        filebeat = self.start_beat()

        # Make sure both files were read
        self.wait_until(
            lambda: self.output_has(lines=3),
            max_timeout=10)

        # Wait until error shows up
        self.wait_until(
            lambda: self.log_contains("Error decoding line: transform: short source buffer"),
            max_timeout=5)

        filebeat.check_kill_and_wait()

        # Check that only 1 registry entry as original was only truncated
        data = self.get_registry()
        assert len(data) == 1

        output = self.read_output_json()
        assert output[2]["message"] == "hello world2"

    def test_debug_reader(self):
        """
        Test that you can enable a debug reader.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
        )

        os.mkdir(self.working_dir + "/log/")

        logfile = self.working_dir + "/log/test.log"

        file = open(logfile, 'w', 0)
        file.write("hello world1")
        file.write("\n")
        file.write("\x00\x00\x00\x00")
        file.write("\n")
        file.write("hello world2")
        file.write("\n")
        file.write("\x00\x00\x00\x00")
        file.write("Hello World\n")
        # Write some more data to hit the 16k min buffer size.
        # Make it web safe.
        file.write(base64.b64encode(os.urandom(16 * 1024)))
        file.close()

        filebeat = self.start_beat()

        # 13 on unix, 14 on windows.
        self.wait_until(lambda: self.log_contains(re.compile(
            'Matching null byte found at offset (13|14)')), max_timeout=5)

        filebeat.check_kill_and_wait()
