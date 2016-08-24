from filebeat import BaseTest

import os
import platform
import time
import shutil
import json
from nose.plugins.skip import Skip, SkipTest


# Additional tests: to be implemented
# * Check if registrar file can be configured, set config param
# * Check "updating" of registrar file
# * Check what happens when registrar file is deleted


class Test(BaseTest):

    def test_registrar_file_content(self):
        """
        Check if registrar file is created correctly and content is as expected
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*"
        )
        os.mkdir(self.working_dir + "/log/")

        # Use \n as line terminator on all platforms per docs.
        line = "hello world\n"
        line_len = len(line) - 1 + len(os.linesep)
        iterations = 5
        testfile = self.working_dir + "/log/test.log"
        file = open(testfile, 'w')
        file.write(iterations * line)
        file.close()

        filebeat = self.start_beat()
        c = self.log_contains_count("states written")

        self.wait_until(
            lambda: self.output_has(lines=5),
            max_timeout=15)

        # Make sure states written appears one more time
        self.wait_until(
            lambda: self.log_contains("states written") > c,
            max_timeout=10)

        # wait until the registry file exist. Needed to avoid a race between
        # the logging and actual writing the file. Seems to happen on Windows.
        self.wait_until(
            lambda: os.path.isfile(os.path.join(self.working_dir,
                                                "registry")),
            max_timeout=1)
        filebeat.check_kill_and_wait()

        # Check that a single file exists in the registry.
        data = self.get_registry()
        assert len(data) == 1

        logFileAbsPath = os.path.abspath(testfile)
        record = self.get_registry_entry_by_path(logFileAbsPath)

        self.assertDictContainsSubset({
            "source": logFileAbsPath,
            "offset": iterations * line_len,
        }, record)
        self.assertTrue("FileStateOS" in record)
        file_state_os = record["FileStateOS"]

        if os.name == "nt":
            # Windows checks
            # TODO: Check for IdxHi, IdxLo, Vol in FileStateOS on Windows.
            self.assertEqual(len(file_state_os), 3)
        elif platform.system() == "SunOS":
            stat = os.stat(logFileAbsPath)
            self.assertEqual(file_state_os["inode"], stat.st_ino)

            # Python does not return the same st_dev value as Golang or the
            # command line stat tool so just check that it's present.
            self.assertTrue("device" in file_state_os)
        else:
            stat = os.stat(logFileAbsPath)
            self.assertDictContainsSubset({
                "inode": stat.st_ino,
                "device": stat.st_dev,
            }, file_state_os)

    def test_registrar_files(self):
        """
        Check that multiple files are put into registrar file
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*"
        )
        os.mkdir(self.working_dir + "/log/")

        testfile1 = self.working_dir + "/log/test1.log"
        testfile2 = self.working_dir + "/log/test2.log"
        file1 = open(testfile1, 'w')
        file2 = open(testfile2, 'w')

        iterations = 5
        for n in range(0, iterations):
            file1.write("hello world")  # 11 chars
            file1.write("\n")  # 1 char
            file2.write("goodbye world")  # 11 chars
            file2.write("\n")  # 1 char

        file1.close()
        file2.close()

        filebeat = self.start_beat()

        self.wait_until(
            lambda: self.output_has(lines=10),
            max_timeout=15)
        # wait until the registry file exist. Needed to avoid a race between
        # the logging and actual writing the file. Seems to happen on Windows.
        self.wait_until(
            lambda: os.path.isfile(os.path.join(self.working_dir,
                                                "registry")),
            max_timeout=1)
        filebeat.check_kill_and_wait()

        # Check that file exist
        data = self.get_registry()

        # Check that 2 files are port of the registrar file
        assert len(data) == 2

    def test_custom_registry_file_location(self):
        """
        Check that when a custom registry file is used, the path
        is created automatically.
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            registryFile="a/b/c/registry",
        )
        os.mkdir(self.working_dir + "/log/")
        testfile = self.working_dir + "/log/test.log"
        with open(testfile, 'w') as f:
            f.write("hello world\n")
        filebeat = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=15)
        # wait until the registry file exist. Needed to avoid a race between
        # the logging and actual writing the file. Seems to happen on Windows.
        self.wait_until(
            lambda: os.path.isfile(os.path.join(self.working_dir,
                                                "a/b/c/registry")),

            max_timeout=1)
        filebeat.check_kill_and_wait()

        assert os.path.isfile(os.path.join(self.working_dir, "a/b/c/registry"))

    def test_rotating_file(self):
        """
        Checks that the registry is properly updated after a file is rotated
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            close_inactive="1s"
        )

        os.mkdir(self.working_dir + "/log/")
        testfile = self.working_dir + "/log/test.log"

        filebeat = self.start_beat()

        with open(testfile, 'w') as f:
            f.write("offset 9\n")

        self.wait_until(lambda: self.output_has(lines=1),
                        max_timeout=10)

        testfilerenamed = self.working_dir + "/log/test.1.log"
        os.rename(testfile, testfilerenamed)

        with open(testfile, 'w') as f:
            f.write("offset 10\n")

        self.wait_until(lambda: self.output_has(lines=2),
                        max_timeout=10)

        # Wait until rotation is detected
        self.wait_until(
            lambda: self.log_contains(
                "Updating state for renamed file"),
            max_timeout=10)

        time.sleep(1)

        filebeat.check_kill_and_wait()

        # Check that file exist
        data = self.get_registry()

        # Make sure the offsets are correctly set
        if os.name == "nt":
            assert self.get_registry_entry_by_path(os.path.abspath(testfile))["offset"] == 11
            assert self.get_registry_entry_by_path(os.path.abspath(testfilerenamed))["offset"] == 10
        else:
            assert self.get_registry_entry_by_path(os.path.abspath(testfile))["offset"] == 10
            assert self.get_registry_entry_by_path(os.path.abspath(testfilerenamed))["offset"] == 9

        # Check that 2 files are port of the registrar file
        assert len(data) == 2

    def test_data_path(self):
        """
        Checks that the registry file is written in a custom data path.
        """
        self.render_config_template(
            path=self.working_dir + "/test.log",
            path_data=self.working_dir + "/datapath",
            skip_registry_config=True,
        )
        with open(self.working_dir + "/test.log", "w") as f:
            f.write("test message\n")
        filebeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=1))
        filebeat.check_kill_and_wait()

        assert os.path.isfile(self.working_dir + "/datapath/registry")

    def test_rotating_file_inode(self):
        """
        Check that inodes are properly written during file rotation
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/input*",
            scan_frequency="1s",
            close_inactive="1s",
        )

        if os.name == "nt":
            raise SkipTest

        os.mkdir(self.working_dir + "/log/")
        testfile = self.working_dir + "/log/input"

        filebeat = self.start_beat()

        with open(testfile, 'w') as f:
            f.write("entry1\n")

        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)

        data = self.get_registry()
        assert os.stat(testfile).st_ino == self.get_registry_entry_by_path(os.path.abspath(testfile))["FileStateOS"]["inode"]

        testfilerenamed1 = self.working_dir + "/log/input.1"
        os.rename(testfile, testfilerenamed1)

        with open(testfile, 'w') as f:
            f.write("entry2\n")

        self.wait_until(
            lambda: self.output_has(lines=2),
            max_timeout=10)

        # Wait until rotation is detected
        self.wait_until(
            lambda: self.log_contains_count(
                "Updating state for renamed file") == 1,
            max_timeout=10)

        time.sleep(1)

        data = self.get_registry()

        assert os.stat(testfile).st_ino == self.get_registry_entry_by_path(os.path.abspath(testfile))["FileStateOS"]["inode"]
        assert os.stat(testfilerenamed1).st_ino == self.get_registry_entry_by_path(os.path.abspath(testfilerenamed1))["FileStateOS"]["inode"]

        # Rotate log file, create a new empty one and remove it afterwards
        testfilerenamed2 = self.working_dir + "/log/input.2"
        os.rename(testfilerenamed1, testfilerenamed2)
        os.rename(testfile, testfilerenamed1)

        with open(testfile, 'w') as f:
            f.write("")

        os.remove(testfilerenamed2)

        with open(testfile, 'w') as f:
            f.write("entry3\n")

        self.wait_until(
            lambda: self.output_has(lines=3),
            max_timeout=10)

        filebeat.check_kill_and_wait()

        data = self.get_registry()

        # Compare file inodes and the one in the registry
        assert os.stat(testfile).st_ino == self.get_registry_entry_by_path(os.path.abspath(testfile))["FileStateOS"]["inode"]
        assert os.stat(testfilerenamed1).st_ino == self.get_registry_entry_by_path(os.path.abspath(testfilerenamed1))["FileStateOS"]["inode"]

        # Check that 3 files are part of the registrar file. The deleted file should never have been detected, but the rotated one should be in
        assert len(data) == 3


    def test_restart_continue(self):
        """
        Check that file readining continues after restart
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/input*",
            scan_frequency="1s"
        )

        if os.name == "nt":
            raise SkipTest

        os.mkdir(self.working_dir + "/log/")
        testfile = self.working_dir + "/log/input"

        filebeat = self.start_beat()

        with open(testfile, 'w') as f:
            f.write("entry1\n")

        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)

        # Wait a momemt to make sure registry is completely written
        time.sleep(1)

        assert os.stat(testfile).st_ino == self.get_registry_entry_by_path(os.path.abspath(testfile))["FileStateOS"]["inode"]

        filebeat.check_kill_and_wait()

        # Store first registry file
        shutil.copyfile(self.working_dir + "/registry", self.working_dir + "/registry.first")

        # Append file
        with open(testfile, 'a') as f:
            f.write("entry2\n")

        filebeat = self.start_beat(output="filebeat2.log")

        # Output file was rotated
        self.wait_until(
            lambda: self.output_has(lines=1, output_file="output/filebeat.1"),
            max_timeout=10)

        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)

        filebeat.check_kill_and_wait()

        data = self.get_registry()

        # Compare file inodes and the one in the registry
        assert os.stat(testfile).st_ino == self.get_registry_entry_by_path(os.path.abspath(testfile))["FileStateOS"]["inode"]

        # Check that 1 files are part of the registrar file. The deleted file should never have been detected
        assert len(data) == 1

        output = self.read_output()

        # Check that output file has the same number of lines as the log file
        assert 1 == len(output)
        assert output[0]["message"] == "entry2"


    def test_rotating_file_with_restart(self):
        """
        Check that inodes are properly written during file rotation and restart
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/input*",
            scan_frequency="1s",
            close_inactive="1s"
        )

        if os.name == "nt":
            raise SkipTest

        os.mkdir(self.working_dir + "/log/")
        testfile = self.working_dir + "/log/input"

        filebeat = self.start_beat()

        with open(testfile, 'w') as f:
            f.write("entry1\n")

        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)

        # Wait a momemt to make sure registry is completely written
        time.sleep(1)

        data = self.get_registry()
        assert os.stat(testfile).st_ino == self.get_registry_entry_by_path(os.path.abspath(testfile))["FileStateOS"]["inode"]

        testfilerenamed1 = self.working_dir + "/log/input.1"
        os.rename(testfile, testfilerenamed1)

        with open(testfile, 'w') as f:
            f.write("entry2\n")

        self.wait_until(
            lambda: self.output_has(lines=2),
            max_timeout=10)

        # Wait until rotation is detected
        self.wait_until(
            lambda: self.log_contains(
                "Updating state for renamed file"),
            max_timeout=10)

        # Wait a momemt to make sure registry is completely written
        time.sleep(1)

        data = self.get_registry()

        assert os.stat(testfile).st_ino == self.get_registry_entry_by_path(os.path.abspath(testfile))["FileStateOS"]["inode"]
        assert os.stat(testfilerenamed1).st_ino == self.get_registry_entry_by_path(os.path.abspath(testfilerenamed1))["FileStateOS"]["inode"]

        filebeat.check_kill_and_wait()

        # Store first registry file
        shutil.copyfile(self.working_dir + "/registry", self.working_dir + "/registry.first")

        # Rotate log file, create a new empty one and remove it afterwards
        testfilerenamed2 = self.working_dir + "/log/input.2"
        os.rename(testfilerenamed1, testfilerenamed2)
        os.rename(testfile, testfilerenamed1)

        with open(testfile, 'w') as f:
            f.write("")

        os.remove(testfilerenamed2)

        with open(testfile, 'w') as f:
            f.write("entry3\n")

        filebeat = self.start_beat(output="filebeat2.log")

        # Output file was rotated
        self.wait_until(
            lambda: self.output_has(lines=2, output_file="output/filebeat.1"),
            max_timeout=10)

        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)

        filebeat.check_kill_and_wait()

        data = self.get_registry()

        # Compare file inodes and the one in the registry
        assert os.stat(testfile).st_ino == self.get_registry_entry_by_path(os.path.abspath(testfile))["FileStateOS"]["inode"]
        assert os.stat(testfilerenamed1).st_ino == self.get_registry_entry_by_path(os.path.abspath(testfilerenamed1))["FileStateOS"]["inode"]

        # Check that 3 files are part of the registrar file. The deleted file should never have been detected, but the rotated one should be in
        assert len(data) == 3

    def test_state_after_rotation(self):
        """
        Checks that the state is written correctly after rotation
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/input*",
            ignore_older="2m",
            scan_frequency="1s",
            close_inactive="1s"
        )

        os.mkdir(self.working_dir + "/log/")
        testfile1 = self.working_dir + "/log/input"
        testfile2 = self.working_dir + "/log/input.1"
        testfile3 = self.working_dir + "/log/input.2"

        with open(testfile1, 'w') as f:
            f.write("entry10\n")

        with open(testfile2, 'w') as f:
            f.write("entry0\n")

        filebeat = self.start_beat()

        self.wait_until(
            lambda: self.output_has(lines=2),
            max_timeout=10)

        # Wait a moment to make sure file exists
        time.sleep(1)
        data = self.get_registry()

        # Check that offsets are correct
        if os.name == "nt":
            # Under windows offset is +1 because of additional newline char
            assert self.get_registry_entry_by_path(os.path.abspath(testfile1))["offset"] == 9
            assert self.get_registry_entry_by_path(os.path.abspath(testfile2))["offset"] == 8
        else:
            assert self.get_registry_entry_by_path(os.path.abspath(testfile1))["offset"] == 8
            assert self.get_registry_entry_by_path(os.path.abspath(testfile2))["offset"] == 7

        # Rotate files and remove old one
        os.rename(testfile2, testfile3)
        os.rename(testfile1, testfile2)

        with open(testfile1, 'w') as f:
            f.write("entry200\n")

        # Remove file afterwards to make sure not inode reuse happens
        os.remove(testfile3)

        # Now wait until rotation is detected
        self.wait_until(
            lambda: self.log_contains(
                "Updating state for renamed file"),
            max_timeout=10)

        self.wait_until(
            lambda: self.log_contains_count(
                "Registry file updated. 2 states written.") >= 1,
            max_timeout=15)

        time.sleep(1)
        filebeat.kill_and_wait()

        # Check that offsets are correct
        if os.name == "nt":
            # Under windows offset is +1 because of additional newline char
            assert self.get_registry_entry_by_path(os.path.abspath(testfile1))["offset"] == 10
            assert self.get_registry_entry_by_path(os.path.abspath(testfile2))["offset"] == 9
        else:
            assert self.get_registry_entry_by_path(os.path.abspath(testfile1))["offset"] == 9
            assert self.get_registry_entry_by_path(os.path.abspath(testfile2))["offset"] == 8


    def test_state_after_rotation_ignore_older(self):
        """
        Checks that the state is written correctly after rotation and ignore older
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/input*",
            ignore_older="2m",
            scan_frequency="1s",
            close_inactive="1s"
        )


        os.mkdir(self.working_dir + "/log/")
        testfile1 = self.working_dir + "/log/input"
        testfile2 = self.working_dir + "/log/input.1"
        testfile3 = self.working_dir + "/log/input.2"

        with open(testfile1, 'w') as f:
            f.write("entry10\n")

        with open(testfile2, 'w') as f:
            f.write("entry0\n")

        # Change modification time so file extends ignore_older
        yesterday = time.time() - 3600*24
        os.utime(testfile2, (yesterday, yesterday))

        filebeat = self.start_beat()

        self.wait_until(
            lambda: self.output_has(lines=1),
            max_timeout=10)

        # Wait a moment to make sure file exists
        time.sleep(1)
        data = self.get_registry()

        # Check that offsets are correct
        if os.name == "nt":
            # Under windows offset is +1 because of additional newline char
            assert self.get_registry_entry_by_path(os.path.abspath(testfile1))["offset"] == 9
        else:
            assert self.get_registry_entry_by_path(os.path.abspath(testfile1))["offset"] == 8

        # Rotate files and remove old one
        os.rename(testfile2, testfile3)
        os.rename(testfile1, testfile2)

        with open(testfile1, 'w') as f:
            f.write("entry200\n")

        # Remove file afterwards to make sure not inode reuse happens
        os.remove(testfile3)

        # Now wait until rotation is detected
        self.wait_until(
            lambda: self.log_contains(
                "Updating state for renamed file"),
            max_timeout=10)

        self.wait_until(
            lambda: self.log_contains_count(
                "Registry file updated. 2 states written.") >= 1,
            max_timeout=15)

        # Wait a momemt to make sure registry is completely written
        time.sleep(1)
        filebeat.kill_and_wait()

        # Check that offsets are correct
        if os.name == "nt":
            # Under windows offset is +1 because of additional newline char
            assert self.get_registry_entry_by_path(os.path.abspath(testfile1))["offset"] == 10
            assert self.get_registry_entry_by_path(os.path.abspath(testfile2))["offset"] == 9
        else:
            assert self.get_registry_entry_by_path(os.path.abspath(testfile1))["offset"] == 9
            assert self.get_registry_entry_by_path(os.path.abspath(testfile2))["offset"] == 8


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
        oldJson = json.loads('{"source":"logs/hello.log","offset":4,"FileStateOS":{"inode":30178938,"device":16777220}}')
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


    def test_clean_inactive(self):
        """
        Checks that states are properly removed after clean_inactive
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/input*",
            clean_inactive="4s",
            ignore_older="2s",
            close_inactive="0.2s",
            scan_frequency="0.1s"
        )

        os.mkdir(self.working_dir + "/log/")
        testfile1 = self.working_dir + "/log/input1"
        testfile2 = self.working_dir + "/log/input2"
        testfile3 = self.working_dir + "/log/input3"

        with open(testfile1, 'w') as f:
            f.write("first file\n")

        with open(testfile2, 'w') as f:
            f.write("second file\n")

        filebeat = self.start_beat()

        self.wait_until(
            lambda: self.output_has(lines=2),
            max_timeout=10)

        # Wait until registry file is created
        self.wait_until(
            lambda: self.log_contains("Registry file updated"),
            max_timeout=15)

        data = self.get_registry()
        assert len(data) == 2

        # Wait until states are removed from prospectors
        self.wait_until(
            lambda: self.log_contains_count(
                "State removed for") == 2,
            max_timeout=15)

        with open(testfile3, 'w') as f:
            f.write("2\n")

        # Write new file to make sure registrar is flushed again
        self.wait_until(
            lambda: self.output_has(lines=3),
            max_timeout=30)

        # Wait until states are removed from prospectors
        self.wait_until(
            lambda: self.log_contains_count(
                "State removed for") == 4,
            max_timeout=15)

        filebeat.check_kill_and_wait()

        # Check that the first to files were removed from the registry
        data = self.get_registry()
        assert len(data) == 1

        # Make sure the last file in the registry is the correct one and has the correct offset
        if os.name == "nt":
            assert data[0]["offset"] == 3
        else:
            assert data[0]["offset"] == 2


    def test_clean_removed(self):
        """
        Checks that files which were removed, the state is removed
        """
        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/input*",
            scan_frequency="0.1s",
            clean_removed=True,
            close_removed=True
        )

        os.mkdir(self.working_dir + "/log/")
        testfile1 = self.working_dir + "/log/input1"
        testfile2 = self.working_dir + "/log/input2"

        with open(testfile1, 'w') as f:
            f.write("file to be removed\n")

        with open(testfile2, 'w') as f:
            f.write("2\n")

        filebeat = self.start_beat()

        self.wait_until(
            lambda: self.output_has(lines=2),
            max_timeout=10)

        # Wait until registry file is created
        self.wait_until(
            lambda: self.log_contains("Registry file updated"),
            max_timeout=15)

        data = self.get_registry()
        assert len(data) == 2

        os.remove(testfile1)

        # Wait until states are removed from prospectors
        self.wait_until(
            lambda: self.log_contains(
                "Remove state for file as file removed"),
            max_timeout=15)

        # Add one more line to make sure registry is written
        with open(testfile2, 'a') as f:
            f.write("make sure registry is written\n")

        self.wait_until(
            lambda: self.output_has(lines=3),
            max_timeout=10)

        filebeat.check_kill_and_wait()

        # Check that the first to files were removed from the registry
        data = self.get_registry()
        assert len(data) == 1

        # Make sure the last file in the registry is the correct one and has the correct offset
        if os.name == "nt":
            assert data[0]["offset"] == len("make sure registry is written\n" + "2\n") + 2
        else:
            assert data[0]["offset"] == len("make sure registry is written\n" + "2\n")
