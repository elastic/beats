from journalbeat import BaseTest

import os
import sys
import unittest
import time


class Test(BaseTest):

    @unittest.skipUnless(sys.platform.startswith("linux"), "Journald only on Linux")
    def test_start_with_local_journal(self):
        """
        Basic test with exiting Journalbeat normally
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*"
        )
        journalbeat_proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("journalbeat is running"))
        exit_code = journalbeat_proc.kill_and_wait()
        print(exit_code)
        assert exit_code == 0

    @unittest.skipUnless(sys.platform.startswith("linux"), "Journald only on Linux")
    def test_start_with_journal_directory(self):
        """
        Basic test with exiting Journalbeat normally
        """

        self.render_config_template(
            journal_path=self.beat_path + "/tests/system/input/",
            path=os.path.abspath(self.working_dir) + "/log/*"
        )
        journalbeat_proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("journalbeat is running"))
        exit_code = journalbeat_proc.kill_and_wait()
        assert exit_code == 0

    @unittest.skipUnless(sys.platform.startswith("linux"), "Journald only on Linux")
    def test_start_with_selected_journal_file(self):
        """
        Basic test with exiting Journalbeat normally
        """

        self.render_config_template(
            journal_path=self.beat_path + "/tests/system/input/test.journal",
            path=os.path.abspath(self.working_dir) + "/log/*"
        )
        journalbeat_proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("journalbeat is running"))
        exit_code = journalbeat_proc.kill_and_wait()
        assert exit_code == 0

#    @unittest.skipUnless(sys.platform.startswith("linux"), "Journald only on Linux")
#    def test_follow_files(self):
#        """
#        Basic test with exiting Journalbeat normally
#        """
#
#        self.render_config_template(
#            journal_path=self.beat_path + "/tests/system/this.journal",
#            path=os.path.abspath(self.working_dir) + "/log/*"
#        )
#
#        journalbeat_proc = self.start_beat()
#        self.wait_until(lambda: self.log_contains("journalbeat is running"))
#        exit_code = journalbeat_proc.kill_and_wait()
#        assert exit_code == 0


if __name__ == '__main__':
    unittest.main()
