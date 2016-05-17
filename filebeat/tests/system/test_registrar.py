from filebeat import TestCase

import os
import platform
import time
import shutil
from nose.plugins.skip import Skip, SkipTest


# Additional tests: to be implemented
# * Check if registrar file can be configured, set config param
# * Check "updating" of registrar file
# * Check what happens when registrar file is deleted


class Test(TestCase):

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

        filebeat = self.start_filebeat()
        c = self.log_contains_count("states written")

        self.wait_until(
                lambda: self.log_contains(
                        "Processing 5 events"),
                max_timeout=15)

        # Make sure states written appears one more time
        self.wait_until(
                lambda: self.log_contains("states written") > c,
                max_timeout=10)

        # wait until the registry file exist. Needed to avoid a race between
        # the logging and actual writing the file. Seems to happen on Windows.
        self.wait_until(
                lambda: os.path.isfile(os.path.join(self.working_dir,
                                                    ".filebeat")),
                max_timeout=1)
        filebeat.kill_and_wait()

        # Check that a single file exists in the registry.
        data = self.get_dot_filebeat()
        assert len(data) == 1

        logFileAbsPath = os.path.abspath(testfile)
        record = data[logFileAbsPath]

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

        filebeat = self.start_filebeat()

        self.wait_until(
                lambda: self.log_contains(
                        "Processing 10 events"),
                max_timeout=15)
        # wait until the registry file exist. Needed to avoid a race between
        # the logging and actual writing the file. Seems to happen on Windows.
        self.wait_until(
                lambda: os.path.isfile(os.path.join(self.working_dir,
                                                    ".filebeat")),
                max_timeout=1)
        filebeat.kill_and_wait()

        # Check that file exist
        data = self.get_dot_filebeat()

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
        filebeat = self.start_filebeat()
        self.wait_until(
                lambda: self.log_contains(
                        "Processing 1 events"),
                max_timeout=15)
        # wait until the registry file exist. Needed to avoid a race between
        # the logging and actual writing the file. Seems to happen on Windows.
        self.wait_until(
                lambda: os.path.isfile(os.path.join(self.working_dir,
                                                    "a/b/c/registry")),

                max_timeout=1)
        filebeat.kill_and_wait()

        assert os.path.isfile(os.path.join(self.working_dir, "a/b/c/registry"))

    def test_rotating_file(self):
        """
        Checks that the registry is properly updated after a file is rotated
        """
        self.render_config_template(
                path=os.path.abspath(self.working_dir) + "/log/*"
        )

        os.mkdir(self.working_dir + "/log/")
        testfile = self.working_dir + "/log/test.log"

        filebeat = self.start_filebeat()

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

        filebeat.kill_and_wait()

        # Check that file exist
        data = self.get_dot_filebeat()

        # Make sure the offsets are correctly set
        data[os.path.abspath(testfile)]["offset"] = 10
        data[os.path.abspath(testfilerenamed)]["offset"] = 9

        # Check that 2 files are port of the registrar file
        assert len(data) == 2

    def test_rotating_file_inode(self):
        """
        Check that inodes are properly written during file rotation
        """
        self.render_config_template(
                path=os.path.abspath(self.working_dir) + "/log/input*",
                scan_frequency="1s"
        )

        if os.name == "nt":
            raise SkipTest

        os.mkdir(self.working_dir + "/log/")
        testfile = self.working_dir + "/log/input"

        filebeat = self.start_filebeat()

        with open(testfile, 'w') as f:
            f.write("entry1\n")

        self.wait_until(
                lambda: self.output_has(lines=1),
                max_timeout=10)

        data = self.get_dot_filebeat()
        assert os.stat(testfile).st_ino == data[os.path.abspath(testfile)]["FileStateOS"]["inode"]

        testfilerenamed1 = self.working_dir + "/log/input.1"
        os.rename(testfile, testfilerenamed1)

        with open(testfile, 'w') as f:
            f.write("entry2\n")

        self.wait_until(
                lambda: self.output_has(lines=2),
                max_timeout=10)

        data = self.get_dot_filebeat()

        assert os.stat(testfile).st_ino == data[os.path.abspath(testfile)]["FileStateOS"]["inode"]
        assert os.stat(testfilerenamed1).st_ino == data[os.path.abspath(testfilerenamed1)]["FileStateOS"]["inode"]

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

        filebeat.kill_and_wait()

        data = self.get_dot_filebeat()

        # Compare file inodes and the one in the registry
        assert os.stat(testfile).st_ino == data[os.path.abspath(testfile)]["FileStateOS"]["inode"]
        assert os.stat(testfilerenamed1).st_ino == data[os.path.abspath(testfilerenamed1)]["FileStateOS"]["inode"]

        # Check that 2 files are part of the registrar file. The deleted file should never have been detected
        assert len(data) == 2


    def test_rotating_file_with_shutdown(self):
        """
        Check that inodes are properly written during file rotation and shutdown
        """
        self.render_config_template(
                path=os.path.abspath(self.working_dir) + "/log/input*",
                scan_frequency="1s"
        )

        if os.name == "nt":
            raise SkipTest

        os.mkdir(self.working_dir + "/log/")
        testfile = self.working_dir + "/log/input"

        filebeat = self.start_filebeat()

        with open(testfile, 'w') as f:
            f.write("entry1\n")

        self.wait_until(
                lambda: self.output_has(lines=1),
                max_timeout=10)

        data = self.get_dot_filebeat()
        assert os.stat(testfile).st_ino == data[os.path.abspath(testfile)]["FileStateOS"]["inode"]

        testfilerenamed1 = self.working_dir + "/log/input.1"
        os.rename(testfile, testfilerenamed1)

        with open(testfile, 'w') as f:
            f.write("entry2\n")

        self.wait_until(
                lambda: self.output_has(lines=2),
                max_timeout=10)

        data = self.get_dot_filebeat()

        assert os.stat(testfile).st_ino == data[os.path.abspath(testfile)]["FileStateOS"]["inode"]
        assert os.stat(testfilerenamed1).st_ino == data[os.path.abspath(testfilerenamed1)]["FileStateOS"]["inode"]

        filebeat.kill_and_wait()

        # Store first registry file
        shutil.copyfile(self.working_dir + "/.filebeat", self.working_dir + "/.filebeat.first")

        # Rotate log file, create a new empty one and remove it afterwards
        testfilerenamed2 = self.working_dir + "/log/input.2"
        os.rename(testfilerenamed1, testfilerenamed2)
        os.rename(testfile, testfilerenamed1)

        with open(testfile, 'w') as f:
            f.write("")

        os.remove(testfilerenamed2)

        with open(testfile, 'w') as f:
            f.write("entry3\n")

        filebeat = self.start_filebeat(output="filebeat2.log")

        # Output file was rotated
        self.wait_until(
                lambda: self.output_has(lines=2, output_file="output/filebeat.1"),
                max_timeout=10)

        self.wait_until(
                lambda: self.output_has(lines=1),
                max_timeout=10)

        filebeat.kill_and_wait()

        data = self.get_dot_filebeat()

        # Compare file inodes and the one in the registry
        assert os.stat(testfile).st_ino == data[os.path.abspath(testfile)]["FileStateOS"]["inode"]
        assert os.stat(testfilerenamed1).st_ino == data[os.path.abspath(testfilerenamed1)]["FileStateOS"]["inode"]

        # Check that 2 files are part of the registrar file. The deleted file should never have been detected
        assert len(data) == 2


    def test_state_after_rotation_ignore_older(self):
        """
        Checks that the state is written correctly after rotation and file under ignore_older before
        """
        self.render_config_template(
                path=os.path.abspath(self.working_dir) + "/log/input*",
                ignoreOlder="2m",
                scan_frequency="1s"
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

        filebeat = self.start_filebeat()

        self.wait_until(
                lambda: self.output_has(lines=1),
                max_timeout=10)

        # Wait a moment to make sure file exists
        time.sleep(1)
        data = self.get_dot_filebeat()

        # Check that offsets are correct
        if os.name == "nt":
            # Under windows offset is +1 because of additional newline char
            assert data[os.path.abspath(testfile1)]["offset"] == 9
            assert data[os.path.abspath(testfile2)]["offset"] == 8
        else:
            assert data[os.path.abspath(testfile1)]["offset"] == 8
            assert data[os.path.abspath(testfile2)]["offset"] == 7


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
                        "File rename was detected, existing file"),
                max_timeout=10)


        c = self.log_contains_count("states written")

        self.wait_until(
                lambda: self.log_contains_count(
                        "Registry file updated. 2 states written.") >= 4,
                max_timeout=15)

        time.sleep(1)
        filebeat.kill_and_wait()


        data = self.get_dot_filebeat()

        # Check that offsets are correct
        if os.name == "nt":
            # Under windows offset is +1 because of additional newline char
            assert data[os.path.abspath(testfile1)]["offset"] == 10
            assert data[os.path.abspath(testfile2)]["offset"] == 9
        else:
            assert data[os.path.abspath(testfile1)]["offset"] == 9
            assert data[os.path.abspath(testfile2)]["offset"] == 8



    def test_state_after_rotation2(self):
        """
        Checks that the state is written correctly after rotation
        """
        self.render_config_template(
                path=os.path.abspath(self.working_dir) + "/log/input*",
                scan_frequency="1s"
        )

        filebeat = self.start_filebeat()


        os.mkdir(self.working_dir + "/log/")
        testfile1 = self.working_dir + "/log/input"
        testfile2 = self.working_dir + "/log/input.1"

        with open(testfile1, 'w') as f:
            f.write("entry10\n")

        self.wait_until(
                lambda: self.output_has(lines=1),
                max_timeout=10)

        # Wait a moment to make sure file exists
        time.sleep(1)
        data = self.get_dot_filebeat()

        # Check that offsets are correct
        if os.name == "nt":
            # Under windows offset is +1 because of additional newline char
            assert data[os.path.abspath(testfile1)]["offset"] == 9
        else:
            assert data[os.path.abspath(testfile1)]["offset"] == 8

        # Rotate files and remove old one
        os.rename(testfile1, testfile2)

        # Now wait until rotation is detected
        self.wait_until(
                lambda: self.log_contains(
                        "File rename was detected, not a new file"),
                max_timeout=10)

        # Wait a moment to make sure file exists
        time.sleep(1)
        data = self.get_dot_filebeat()

        # Check that offsets are correct
        if os.name == "nt":
            # Under windows offset is +1 because of additional newline char
            assert data[os.path.abspath(testfile2)]["offset"] == 9
        else:
            assert data[os.path.abspath(testfile2)]["offset"] == 8


        with open(testfile1, 'w') as f:
            f.write("entry200\n")

        self.wait_until(
                lambda: self.output_has(lines=2),
                max_timeout=10)

        filebeat.kill_and_wait()

        # Wait a moment to make sure file exists
        data = self.get_dot_filebeat()

        # Check that offsets are correct
        if os.name == "nt":
            # Under windows offset is +1 because of additional newline char
            assert data[os.path.abspath(testfile1)]["offset"] == 10
            assert data[os.path.abspath(testfile2)]["offset"] == 9
        else:
            assert data[os.path.abspath(testfile1)]["offset"] == 9
            assert data[os.path.abspath(testfile2)]["offset"] == 8



