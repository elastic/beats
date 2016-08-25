from filebeat import BaseTest
import os
import time

"""
Test Harvesters
"""


class Test(BaseTest):

    def test_close_renamed(self):
        """
        Checks that a file is closed when its renamed / rotated
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/test.log",
            close_renamed="true",
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

        filebeat.check_kill_and_wait()

        data = self.get_registry()

        # Make sure new file was picked up. As it has the same file name,
        # one entry for the new and one for the old should exist
        assert len(data) == 2


    def test_close_removed(self):
        """
        Checks that a file is closed if removed
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/test.log",
            close_removed="true",
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

        # Wait until error shows up on windows
        self.wait_until(
            lambda: self.log_contains(
                "Closing because close_removed is enabled"),
            max_timeout=15)

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

        # Wait until state is written
        self.wait_until(
            lambda: self.log_contains(
                "1 states written."),
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
