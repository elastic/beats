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

        self.wait_until(lambda: self.log_contains("journalbeat is running"), max_timeout=10)

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

        self.wait_until(lambda: self.log_contains("journalbeat is running"), max_timeout=10)
        self.wait_until(lambda: self.log_contains("Tailing the journal file") == 1, max_timeout=10)

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

        self.wait_until(lambda: self.log_contains("journalbeat is running"))
        self.wait_until(
            lambda: self.log_contains("Reading from the beginning of the journal file") == 1,
            max_timeout=10)
        self.wait_until(
            lambda: self.log_contains("\"message\": \"thinkpad_acpi: unhandled HKEY event 0x60b0\"") == 1,
            max_timeout=10)

        exit_code = journalbeat_proc.kill_and_wait()
        assert exit_code == 0


if __name__ == '__main__':
    unittest.main()
