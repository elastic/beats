from filebeat import BaseTest

import os
import platform


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
        self.wait_until(
            lambda: self.log_contains(
                "Processing 5 events"),
            max_timeout=15)

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

        filebeat = self.start_beat()

        self.wait_until(
            lambda: self.log_contains(
                "Processing 10 events"),
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
            lambda: self.log_contains(
                "Processing 1 events"),
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
            path=os.path.abspath(self.working_dir) + "/log/*"
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

        filebeat.check_kill_and_wait()

        # Check that file exist
        data = self.get_registry()

        # Make sure the offsets are correctly set
        data[os.path.abspath(testfile)]["offset"] = 10
        data[os.path.abspath(testfilerenamed)]["offset"] = 9

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
