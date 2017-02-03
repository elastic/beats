from filebeat import BaseTest

import os
import platform
import time
import shutil
import json
import stat
from nose.plugins.skip import Skip, SkipTest


class Test(BaseTest):

    def test_migration_non_windows(self):
        """
        Tests if migration from old filebeat registry to new format works
        """

        if os.name == "nt":
            raise SkipTest

        registry_file = self.working_dir + '/registry'

        # Write old registry file
        with open(registry_file, 'w') as f:
            f.write('{"logs/hello.log":{"source":"logs/hello.log","offset":4,"FileStateOS":{"inode":30178938,"device":16777220}},"logs/log2.log":{"source":"logs/log2.log","offset":6,"FileStateOS":{"inode":30178958,"device":16777220}}}')

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/input*",
            clean_removed="false",
            clean_inactive="0",
        )

        filebeat = self.start_beat()

        self.wait_until(
            lambda: self.log_contains("Old registry states found: 2"),
            max_timeout=15)

        self.wait_until(
            lambda: self.log_contains("Old states converted to new states and written to registrar: 2"),
            max_timeout=15)

        filebeat.check_kill_and_wait()

        # Check if content is same as above
        assert self.get_registry_entry_by_path("logs/hello.log")["offset"] == 4
        assert self.get_registry_entry_by_path("logs/log2.log")["offset"] == 6

        # Compare first entry
        oldJson = json.loads(
            '{"source":"logs/hello.log","offset":4,"FileStateOS":{"inode":30178938,"device":16777220}}')
        newJson = self.get_registry_entry_by_path("logs/hello.log")
        del newJson["timestamp"]
        del newJson["ttl"]
        assert newJson == oldJson

        # Compare second entry
        oldJson = json.loads('{"source":"logs/log2.log","offset":6,"FileStateOS":{"inode":30178958,"device":16777220}}')
        newJson = self.get_registry_entry_by_path("logs/log2.log")
        del newJson["timestamp"]
        del newJson["ttl"]
        assert newJson == oldJson

        # Make sure the right number of entries is in
        data = self.get_registry()
        assert len(data) == 2

    def test_migration_windows(self):
        """
        Tests if migration from old filebeat registry to new format works
        """

        if os.name != "nt":
            raise SkipTest

        registry_file = self.working_dir + '/registry'

        # Write old registry file
        with open(registry_file, 'w') as f:
            f.write('{"logs/hello.log":{"source":"logs/hello.log","offset":4,"FileStateOS":{"idxhi":1,"idxlo":12,"vol":34}},"logs/log2.log":{"source":"logs/log2.log","offset":6,"FileStateOS":{"idxhi":67,"idxlo":44,"vol":12}}}')

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/input*",
        )

        filebeat = self.start_beat()

        self.wait_until(
            lambda: self.log_contains("Old registry states found: 2"),
            max_timeout=15)

        self.wait_until(
            lambda: self.log_contains("Old states converted to new states and written to registrar: 2"),
            max_timeout=15)

        filebeat.check_kill_and_wait()

        # Check if content is same as above
        assert self.get_registry_entry_by_path("logs/hello.log")["offset"] == 4
        assert self.get_registry_entry_by_path("logs/log2.log")["offset"] == 6

        # Compare first entry
        oldJson = json.loads('{"source":"logs/hello.log","offset":4,"FileStateOS":{"idxhi":1,"idxlo":12,"vol":34}}')
        newJson = self.get_registry_entry_by_path("logs/hello.log")
        del newJson["timestamp"]
        del newJson["ttl"]
        assert newJson == oldJson

        # Compare second entry
        oldJson = json.loads('{"source":"logs/log2.log","offset":6,"FileStateOS":{"idxhi":67,"idxlo":44,"vol":12}}')
        newJson = self.get_registry_entry_by_path("logs/log2.log")
        del newJson["timestamp"]
        del newJson["ttl"]
        assert newJson == oldJson

        # Make sure the right number of entries is in
        data = self.get_registry()
        assert len(data) == 2

    def test_migration_continue_reading(self):
        """
        Tests if after the migration filebeat keeps reading the file
        """

        os.mkdir(self.working_dir + "/log/")
        testfile1 = self.working_dir + "/log/test.log"

        with open(testfile1, 'w') as f:
            f.write("entry10\n")

        registry_file = self.working_dir + '/registry'

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            output_file_filename="filebeat_1",
        )

        # Run filebeat to create a registry
        filebeat = self.start_beat(output="filebeat1.log")
        self.wait_until(
            lambda: self.output_has(lines=1, output_file="output/filebeat_1"),
            max_timeout=10)
        filebeat.check_kill_and_wait()

        # Create old registry file out of the new one
        r = self.get_registry()
        registry_entry = r[0]
        del registry_entry["timestamp"]
        del registry_entry["ttl"]
        old_registry = {registry_entry["source"]: registry_entry}

        # Overwrite registry
        with open(registry_file, 'w') as f:
            json.dump(old_registry, f)

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            output_file_filename="filebeat_2",
        )

        filebeat = self.start_beat(output="filebeat2.log")

        # Wait until state is migrated
        self.wait_until(
            lambda: self.log_contains(
                "Old states converted to new states and written to registrar: 1", "filebeat2.log"),
            max_timeout=10)

        with open(testfile1, 'a') as f:
            f.write("entry12\n")

        # After restart new output file is created -> only 1 new entry
        self.wait_until(
            lambda: self.output_has(lines=1, output_file="output/filebeat_2"),
            max_timeout=10)

        filebeat.check_kill_and_wait()
