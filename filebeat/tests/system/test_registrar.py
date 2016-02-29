from filebeat import BaseTest

import os

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

        testfile = self.working_dir + "/log/test.log"
        file = open(testfile, 'w')

        iterations = 5
        file.write(iterations * "hello world\n")

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
                                                ".filebeat")),
            max_timeout=1)
        filebeat.check_kill_and_wait()

        # Check that file exist
        data = self.get_dot_filebeat()

        # Check that offset is set correctly
        logFileAbs = os.path.abspath(testfile)
        # Hello world text plus newline, multiplied by the number
        # of lines and the windows correction applied
        assert data[logFileAbs]['offset'] == \
            iterations * (11 + len(os.linesep))

        # Check that right source field is inside
        assert data[logFileAbs]['source'] == logFileAbs

        # Check that no additional info is in the file
        assert len(data) == 1

        # Windows checks
        if os.name == "nt":

            # TODO: Check for IdxHi, IdxLo, Vol

            assert len(data[logFileAbs]) == 3
            assert len(data[logFileAbs]['FileStateOS']) == 3

        else:
            # Check that inode is set correctly
            inode = os.stat(logFileAbs).st_ino
            assert data[logFileAbs]['FileStateOS']['inode'] == inode

            # Check that device is set correctly
            device = os.stat(logFileAbs).st_dev
            assert os.name == "nt" or \
                data[logFileAbs]['FileStateOS']['device'] == device

            assert len(data[logFileAbs]) == 3
            assert len(data[logFileAbs]['FileStateOS']) == 2

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
                                                ".filebeat")),
            max_timeout=1)
        filebeat.check_kill_and_wait()

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

        self.wait_until(
                lambda: self.output_has(lines=1),
                max_timeout=10)


        testfilerenamed = self.working_dir + "/log/test.1.log"
        os.rename(testfile, testfilerenamed)

        with open(testfile, 'w') as f:
            f.write("offset 10\n")


        self.wait_until(
                lambda: self.output_has(lines=2),
                max_timeout=10)

        filebeat.check_kill_and_wait()

        # Check that file exist
        data = self.get_dot_filebeat()

        # Make sure the offsets are correctly set
        data[os.path.abspath(testfile)]["offset"] = 10
        data[os.path.abspath(testfilerenamed)]["offset"] = 9

        # Check that 2 files are port of the registrar file
        assert len(data) == 2
