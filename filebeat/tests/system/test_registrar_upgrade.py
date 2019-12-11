#!/usr/bin/env python
"""Test the registrar with old registry file formats"""

import os
import json

from nose.plugins.skip import Skip, SkipTest

from filebeat import BaseTest


class Test(BaseTest):
    def test_upgrade_from_6_3_0(self):
        template = "test-2lines-registry-6.3.0"
        self.run_with_single_registry_format(template)

    def test_upgrade_from_6_3_1(self):
        template = "test-2lines-registry-6.3.1"
        self.run_with_single_registry_format(template)

    def test_upgrade_from_faulty_6_3_1(self):
        template = "test-2lines-registry-6.3.1-faulty"
        self.run_with_single_registry_format(template)

    def test_upgrade_from_latest(self):
        template = "test-2lines-registry-latest"
        self.run_with_single_registry_format(template)

    def test_upgrade_from_single_file_to_folder_hierarchy(self):
        template = "test-2lines-registry-latest"
        self.run_with_single_registry_format(template)
        self.validate_if_registry_is_moved_under_folder()

    def run_with_single_registry_format(self, template):
        # prepare log file
        testfile, file_state = self.prepare_log()

        # prepare registry file
        self.apply_registry_template(template, testfile, file_state)

        self.run_and_validate()

    def apply_registry_template(self, template, testfile, file_state):
        source = self.beat_path + "/tests/files/registry/" + template
        with open(source) as f:
            registry = json.loads(f.read())

        for state in registry:
            state["source"] = testfile
            state["FileStateOS"] = file_state
        with open(self.working_dir + "/registry", 'w') as f:
            f.write(json.dumps(registry))

    def prepare_log(self):
        # test is current skipped on windows, due to FileStateOS must match the
        # current OS format.
        if os.name == "nt":
            raise SkipTest

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*"
        )

        os.mkdir(self.working_dir + "/log/")

        testfile_path = self.working_dir + "/log/test.log"
        with open(testfile_path, 'w') as f:
            f.write("123456789\n")
            f.write("abcdefghi\n")

        st = os.stat(testfile_path)
        file_state = {"inode": st.st_ino, "device": st.st_dev}
        return testfile_path, file_state

    def run_and_validate(self):
        filebeat = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=15)

        # stop filebeat and enforce one last registry update
        filebeat.check_kill_and_wait()

        data = self.get_registry()
        assert len(data) == 1
        assert data[0]["offset"] == 20

        # check only second line has been written
        output = self.read_output()
        assert len(output) == 1
        assert output[0]["message"] == "abcdefghi"

    def validate_if_registry_is_moved_under_folder(self):
        migrated_registry_dir = os.path.abspath(self.working_dir + "/registry")
        assert os.path.isdir(migrated_registry_dir)
        assert os.path.isdir(migrated_registry_dir + "/filebeat")
        assert os.path.isfile(migrated_registry_dir + "/filebeat/data.json")
