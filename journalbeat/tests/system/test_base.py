from journalbeat import BaseTest

import os
import sys
import unittest
import time


class Test(BaseTest):

    @unittest.skipUnless(sys.platform.startswith("linux"), "Journald only on Linux")
    def test_start_with_local_journal(self):
        """
        Journalbeat is able to start with the local journal.
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*"
        )
        journalbeat_proc = self.start_beat()

        self.wait_until(lambda: self.log_contains(
            "journalbeat is running"), max_timeout=10)

        exit_code = journalbeat_proc.kill_and_wait()
        assert exit_code == 0

    @unittest.skipUnless(sys.platform.startswith("linux"), "Journald only on Linux")
    def test_start_with_journal_directory(self):
        """
        Journalbeat is able to open a directory of journal files and starts tailing them.
        """

        self.render_config_template(
            journal_path=self.beat_path + "/tests/system/input/",
            path=os.path.abspath(self.working_dir) + "/log/*"
        )
        journalbeat_proc = self.start_beat()

        required_log_snippets = [
            # journalbeat can be started
            "journalbeat is running",
            # journalbeat can seek to the position defined in the cursor
            "Tailing the journal file",
        ]
        for snippet in required_log_snippets:
            self.wait_until(lambda: self.log_contains(snippet),
                            name="Line in '{}' Journalbeat log".format(snippet))

        exit_code = journalbeat_proc.kill_and_wait()
        assert exit_code == 0

    @unittest.skipUnless(sys.platform.startswith("linux"), "Journald only on Linux")
    def test_start_with_selected_journal_file(self):
        """
        Journalbeat is able to open a journal file and start to read it from the begining.
        """

        self.render_config_template(
            journal_path=self.beat_path + "/tests/system/input/test.journal",
            seek_method="head",
            path=os.path.abspath(self.working_dir) + "/log/*"
        )
        journalbeat_proc = self.start_beat()

        required_log_snippets = [
            # journalbeat can be started
            "journalbeat is running",
            # journalbeat can seek to the position defined in the cursor
            "Reading from the beginning of the journal file",
            # message can be read from test journal
            "\"message\": \"thinkpad_acpi: unhandled HKEY event 0x60b0\"",
        ]
        for snippet in required_log_snippets:
            self.wait_until(lambda: self.log_contains(snippet),
                            name="Line in '{}' Journalbeat log".format(snippet))

        exit_code = journalbeat_proc.kill_and_wait()
        assert exit_code == 0

    @unittest.skipUnless(sys.platform.startswith("linux"), "Journald only on Linux")
    def test_read_events_with_existing_registry(self):
        """
        Journalbeat is able to follow reading a from a journal with an existing registry file.
        """

        self.render_config_template(
            journal_path=self.beat_path + "/tests/system/input/test.journal",
            seek_method="cursor",
            registry_file=self.beat_path + "/tests/system/input/test.registry",
            path=os.path.abspath(self.working_dir) + "/log/*",
        )
        journalbeat_proc = self.start_beat()

        required_log_snippets = [
            # journalbeat can be started
            "journalbeat is running",
            # journalbeat can seek to the position defined in the cursor
            "Seeked to position defined in cursor",
            # message can be read from test journal
            "please report the conditions when this event happened to",
            # only one event is read and published
            "journalbeat successfully published 1 events",
        ]
        for snippet in required_log_snippets:
            self.wait_until(lambda: self.log_contains(snippet),
                            name="Line in '{}' Journalbeat log".format(snippet))

        exit_code = journalbeat_proc.kill_and_wait()
        assert exit_code == 0


if __name__ == '__main__':
    unittest.main()
