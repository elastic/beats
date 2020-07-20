from journalbeat import BaseTest

import os
import sys
import unittest
import time
import yaml
from shutil import copyfile
from beat import common_tests


class Test(BaseTest, common_tests.TestExportsMixin):

    @unittest.skipUnless(sys.platform.startswith("linux"), "Journald only on Linux")
    def test_start_with_local_journal(self):
        """
        Journalbeat is able to start with the local journal.
        """

        self.render_config_template(
            inputs=[{
                "paths": [],
            }],
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
            inputs=[{
                "paths": [
                    self.beat_path + "/tests/system/input/",
                ],
                "seek": "tail",
            }],
        )
        journalbeat_proc = self.start_beat()

        self.wait_until(lambda: self.log_contains("journalbeat is running"))

        exit_code = journalbeat_proc.kill_and_wait()
        assert exit_code == 0

        # journalbeat is tailing an inactive journal
        assert self.output_is_empty()

    @unittest.skipUnless(sys.platform.startswith("linux"), "Journald only on Linux")
    def test_start_with_selected_journal_file(self):
        """
        Journalbeat is able to open a journal file and start to read it from the begining.
        """

        self.render_config_template(
            inputs=[{
                "paths": [
                    self.beat_path + "/tests/system/input/test.journal",
                ],
                "seek": "head",
            }],
        )
        journalbeat_proc = self.start_beat()

        self.wait_until(lambda: self.output_has(lines=23))

        exit_code = journalbeat_proc.kill_and_wait()
        assert exit_code == 0

    @unittest.skipUnless(sys.platform.startswith("linux"), "Journald only on Linux")
    def test_start_with_selected_journal_file_with_cursor_fallback(self):
        """
        Journalbeat is able to open a journal file and start to read it from the position configured by seek and cursor_seek_fallback.
        """

        self.render_config_template(
            inputs=[{
                "paths": [
                    self.beat_path + "/tests/system/input/test.journal",
                ],
                "seek": "cursor",
                "cursor_seek_fallback": "tail",
            }],
        )
        journalbeat_proc = self.start_beat()

        self.wait_until(lambda: self.log_contains("journalbeat is running"))

        exit_code = journalbeat_proc.kill_and_wait()
        assert exit_code == 0

        # journalbeat is tailing an inactive journal with no cursor data
        assert self.output_is_empty()

    @unittest.skipUnless(sys.platform.startswith("linux"), "Journald only on Linux")
    def test_read_events_with_existing_registry(self):
        """
        Journalbeat is able to follow reading a from a journal with an existing registry file.
        """

        registry_path = os.path.join(os.path.abspath(self.working_dir), "data", "registry")
        os.mkdir(os.path.dirname(registry_path))
        copyfile(self.beat_path + "/tests/system/input/test.registry",
                 os.path.join(os.path.abspath(self.working_dir), "data/registry"))
        input_path = self.beat_path + "/tests/system/input/test.journal"
        self._prepare_registry_file(registry_path, input_path)

        self.render_config_template(
            inputs=[{
                "paths": [input_path],
                "seek": "cursor",
                "cursor_seek_fallback": "tail",
            }],
        )
        journalbeat_proc = self.start_beat()

        self.wait_until(lambda: self.output_has(lines=9))

        exit_code = journalbeat_proc.kill_and_wait()
        assert exit_code == 0

    @unittest.skipUnless(sys.platform.startswith("linux"), "Journald only on Linux")
    def test_read_events_with_include_matches(self):
        """
        Journalbeat is able to pass matchers to the journal reader and read filtered messages.
        """

        self.render_config_template(
            inputs=[{
                "paths": [
                    self.beat_path + "/tests/system/input/test.journal",
                ],
                "seek": "head",
                "include_matches": [
                    "syslog.priority=6",
                ]
            }],
        )
        journalbeat_proc = self.start_beat()

        self.wait_until(lambda: self.output_has(lines=6))

        exit_code = journalbeat_proc.kill_and_wait()
        assert exit_code == 0

    @unittest.skipUnless(sys.platform.startswith("linux"), "Journald only on Linux")
    def test_input_id(self):
        """
        Journalbeat persists states with IDs.
        """

        self.render_config_template(
            inputs=[
                {
                    "id": "serviceA.unit",
                    "paths": [
                        self.beat_path + "/tests/system/input/test.journal",
                    ],
                },
                {
                    "id": "serviceB unit",
                    "paths": [
                        self.beat_path + "/tests/system/input/test.journal",
                    ],
                }
            ],
        )

        # Run the beat until it publishes events from both inputs.
        journalbeat_proc = self.start_beat()
        expected_msg = 'successfully published events'
        self.wait_until(lambda: self.log_contains(expected_msg))
        self.wait_until(lambda: self.log_contains(expected_msg))
        journalbeat_proc.check_kill_and_wait()

        # Verify that registry paths are prefixed with an ID.
        registry_data = self.read_registry()
        self.assertIn("journal_entries", registry_data)
        journal_entries = registry_data['journal_entries']
        self.assertGreater(len(journal_entries), 0)
        for item in journal_entries:
            self.assertTrue(item['path'].startswith('journald::'), "starts with journald::")
            self.assertTrue(item['path'].find('::service'), "ends with ::<id>")

    def _prepare_registry_file(self, registry_path, journal_path):
        lines = []
        with open(registry_path, "r") as registry_file:
            lines = registry_file.readlines()
            lines[2] = "- path: " + journal_path + "\n"

        with open(registry_path, "w") as registry_file:
            for line in lines:
                registry_file.write(line)

    def read_registry(self):
        registry_path = os.path.join(os.path.abspath(self.working_dir), "data", "registry")

        with open(registry_path, "r") as stream:
            return yaml.safe_load(stream)


if __name__ == '__main__':
    unittest.main()
