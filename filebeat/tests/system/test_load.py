from filebeat import BaseTest
import os
import logging
import logging.handlers
import json
from nose.plugins.skip import Skip, SkipTest
import time

"""
Test filebeat under different load scenarios
"""


class Test(BaseTest):
    def test_no_missing_events(self):
        """
        Test that filebeat does not loose any events under heavy file rotation and load
        """

        if os.name == "nt":
            # This test is currently skipped on windows because very fast file
            # rotation cannot happen when harvester has file handler still open.
            raise SkipTest

        log_file = self.working_dir + "/log/test.log"
        os.mkdir(self.working_dir + "/log/")

        logger = logging.getLogger('beats-logger')
        total_lines = 1000
        lines_per_file = 10
        # Each line should have the same length + line ending
        # Some spare capacity is added to make sure all events are presisted
        line_length = len(str(total_lines)) + 1

        # Setup python log handler
        handler = logging.handlers.RotatingFileHandler(
            log_file, maxBytes=line_length * lines_per_file + 1,
            backupCount=total_lines / lines_per_file + 1)
        logger.addHandler(handler)

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            rotate_every_kb=(total_lines * (line_length +1)),    # With filepath, each line can be up to 1KB is assumed
        )

        # Start filebeat
        filebeat = self.start_beat()

        # wait until filebeat is fully running
        self.wait_until(
            lambda: self.log_contains("All prospectors are initialised and running"),
            max_timeout=15)

        # Start logging and rotating
        for i in range(total_lines):
            # Make sure each line has the same length
            line = format(i, str(line_length - 1))
            logger.debug("%d", i)

        # wait until all lines are read
        self.wait_until(
            lambda: self.output_has(lines=total_lines),
            max_timeout=15)

        filebeat.check_kill_and_wait()

        entry_list = []

        with open(self.working_dir + "/output/filebeat") as f:
            for line in f:
                content = json.loads(line)
                v = int(content["message"])
                entry_list.append(v)

        ### This lines can be uncomemnted for debugging ###
        # Prints out the missing entries
        #for i in range(total_lines):
        #    if i not in entry_list:
        #        print i
        # Stats about the files read
        #unique_entries = len(set(entry_list))
        #print "Total lines: " + str(total_lines)
        #print "Total unique entries: " + str(unique_entries)
        #print "Total entries: " + str(len(entry_list))
        #print "Registry entries: " + str(len(data))

        # Check that file exist
        data = self.get_registry()

        paths = os.listdir(self.working_dir + "/log/")
        assert len(paths) == len(data)

        for i in range(total_lines):
            assert i in entry_list

        # Compares unique entries
        assert len(set(entry_list)) == total_lines
        assert len(entry_list) == total_lines


    def test_no_open_files_left(self):
        """
        Test that filebeat not keep any files open
        """

        # This is not implemented yet, here as a reminder
        raise SkipTest

