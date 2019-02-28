from journalbeat import BaseTest

import os
import sys
import unittest
import time
import yaml


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

        self.wait_until(lambda: self.log_contains("journalbeat is running"))

        exit_code = journalbeat_proc.kill_and_wait()
        assert exit_code == 0

    @unittest.skipUnless(sys.platform.startswith("linux"), "Journald only on Linux")
    def test_start_with_journal_directory(self):
        """
        Journalbeat is able to open a directory of journal files and starts tailing them.
        """

        self.render_config_template(
            journal_path=self.beat_path + "/tests/system/input/",
            seek_method="tail",
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
    def test_start_with_selected_journal_file_with_cursor_fallback(self):
        """
        Journalbeat is able to open a journal file and start to read it from the position configured by seek and cursor_seek_fallback.
        """

        self.render_config_template(
            journal_path=self.beat_path + "/tests/system/input/test.journal",
            seek_method="cursor",
            cursor_seek_fallback="tail",
            path=os.path.abspath(self.working_dir) + "/log/*"
        )
        journalbeat_proc = self.start_beat()

        required_log_snippets = [
            # journalbeat can be started
            "journalbeat is running",
            # journalbeat can seek to the position defined in cursor_seek_fallback.
            "Seeking method set to cursor, but no state is saved for reader. Starting to read from the end",
            # message can be read from test journal
            "\"message\": \"thinkpad_acpi: please report the conditions when this event happened to ibm-acpi-devel@lists.sourceforge.net\"",
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

        registry_path = self.beat_path + "/tests/system/input/test.registry"
        input_path = self.beat_path + "/tests/system/input/test.journal"
        self._prepare_registry_file(registry_path, input_path)

        self.render_config_template(
            journal_path=input_path,
            seek_method="cursor",
            registry_file=registry_path,
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

    @unittest.skipUnless(sys.platform.startswith("linux"), "Journald only on Linux")
    def test_read_events_with_include_matches(self):
        """
        Journalbeat is able to pass matchers to the journal reader and read filtered messages.
        """

        self.render_config_template(
            journal_path=self.beat_path + "/tests/system/input/test.journal",
            seek_method="head",
            matches="syslog.priority=5",
            path=os.path.abspath(self.working_dir) + "/log/*",
        )
        journalbeat_proc = self.start_beat()

        required_log_snippets = [
            # journalbeat can be started
            "journalbeat is running",
            # message can be read from test journal
            "unhandled HKEY event 0x60b1",
            "please report the conditions when this event happened to",
            # Four events with priority 5 is publised
            "journalbeat successfully published 5 events",
        ]
        for snippet in required_log_snippets:
            self.wait_until(lambda: self.log_contains(snippet),
                            name="Line in '{}' Journalbeat log".format(snippet))

        exit_code = journalbeat_proc.kill_and_wait()
        assert exit_code == 0

    @unittest.skipUnless(sys.platform.startswith("linux"), "Journald only on Linux")
    def test_read_events_with_syslog_identifier_filtering(self):
        """
        Journalbeat is able to pass matchers to the journal reader to test identifiers filter.
        """

        self.render_config_template(
            journal_path=self.beat_path + "/tests/system/input/bigger.journal",
            seek_method="head",
            identifiers="audit",
            path=os.path.abspath(self.working_dir) + "/log/*",
        )
        journalbeat_proc = self.start_beat()

        required_log_snippets = [
            # journalbeat can be started
            "journalbeat is running",
            # message can be read from test journal
	    "SECCOMP auid=1000 uid=1000 gid=1000 ses=1 pid=20166",
	    "SECCOMP auid=1000 uid=1000 gid=1000 ses=1 pid=20166",
	    "SECCOMP auid=1000 uid=1000 gid=1000 ses=1 pid=20285",
	    "SECCOMP auid=1000 uid=1000 gid=1000 ses=1 pid=21882",
	    "SECCOMP auid=1000 uid=1000 gid=1000 ses=1 pid=21882",
	    "SECCOMP auid=1000 uid=1000 gid=1000 ses=1 pid=19234",
	    "SECCOMP auid=1000 uid=1000 gid=1000 ses=1 pid=17757",

            # 25 events with priority 5 is publised
            "journalbeat successfully published 26 events",
        ]
        for snippet in required_log_snippets:
            self.wait_until(lambda: self.log_contains(snippet),
                            name="Line in '{}' Journalbeat log".format(snippet))

        exit_code = journalbeat_proc.kill_and_wait()
        assert exit_code == 0

    @unittest.skipUnless(sys.platform.startswith("linux"), "Journald only on Linux")
    def test_read_events_with_sytemd_unit_filtering(self):
        """
        Journalbeat is able to pass matchers to the journal reader to test units filters.
        """

        self.render_config_template(
            journal_path=self.beat_path + "/tests/system/input/bigger.journal",
            seek_method="head",
            units="docker.service",
            path=os.path.abspath(self.working_dir) + "/log/*",
        )
        journalbeat_proc = self.start_beat()

        required_log_snippets = [
            # journalbeat can be started
            "journalbeat is running",
            # message can be read from test journal
	    "Ignoring Exit Event, no such exec command found",
	    "Health check for container a6f209d75876b7d113ee88abb19966e2cd6aa74eeb0d4e1fffc6856133d6786a",
	    "Health check for container 9fbde5825e1b45aa34a221a992611c323ac6bcc96eb7eeb30da1a62015b89683",

            # 25 events with priority 5 is publised
            "journalbeat successfully published 26 events",
        ]
        for snippet in required_log_snippets:
            self.wait_until(lambda: self.log_contains(snippet),
                            name="Line in '{}' Journalbeat log".format(snippet))

        exit_code = journalbeat_proc.kill_and_wait()
        assert exit_code == 0

    def _prepare_registry_file(self, registry_path, journal_path):
        lines = []
        with open(registry_path, "r") as registry_file:
            lines = registry_file.readlines()
            lines[2] = "- path: " + journal_path + "\n"

        with open(registry_path, "w") as registry_file:
            for line in lines:
                registry_file.write(line)



if __name__ == '__main__':
    unittest.main()
