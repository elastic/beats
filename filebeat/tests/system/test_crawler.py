# -*- coding: utf-8 -*-
import codecs
import os
import shutil
import time
import unittest
from filebeat import BaseTest

# Additional tests to be added:
# * Check what happens when file renamed -> no recrawling should happen
# * Check if file descriptor is "closed" when file disappears


class Test(BaseTest):

    def test_file_disappear_appear(self):
        """
        Checks that filebeat keeps running in case a log files is deleted

        On Windows this tests in addition if the file was closed as it couldn't be found anymore
        If Windows does not close the file, a new one can't be created.
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*.log",
            close_removed="true",
            scan_frequency="0.1s",
            clean_removed="false",
        )
        os.mkdir(self.working_dir + "/log/")

        testfile = self.working_dir + "/log/test.log"
        testfilenew = self.working_dir + "/log/hiddenfile"
        file = open(testfile, 'w')

        # Creates testfile now, to prevent inode reuse
        open(testfilenew, 'a').close()

        iterations1 = 5
        for n in range(0, iterations1):
            file.write("disappearing file")
            file.write("\n")

        file.close()

        filebeat = self.start_beat()

        # Let it read the file
        self.wait_until(
            lambda: self.output_has(lines=iterations1), max_timeout=10)
        os.remove(testfile)

        # Wait until error shows up on windows
        self.wait_until(
            lambda: self.log_contains(
                "Closing because close_removed is enabled"),
            max_timeout=15)

        # Move file to old file name
        shutil.move(testfilenew, testfile)
        file = open(testfile, 'w')

        iterations2 = 6
        for n in range(0, iterations2):
            file.write("new file")
            file.write("\n")

        file.close()

        # Let it read the file
        self.wait_until(
            lambda: self.output_has(
                lines=iterations1 +
                iterations2),
            max_timeout=10)

        filebeat.check_kill_and_wait()

        data = self.get_registry()

        # Make sure new file was picked up. As it has the same file name,
        # one entry for the new file and one for the old should exist
        assert len(data) == 2

        # Make sure output has 11 entries, the new file was started
        # from scratch
        output = self.read_output()
        assert len(output) == 5 + 6

    def test_multiple_appends(self):
        """
        Test that filebeat keeps picking up new lines
        after appending multiple times
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
        )
        os.mkdir(self.working_dir + "/log/")

        testfile = self.working_dir + "/log/test.log"

        filebeat = self.start_beat()

        # Write initial file
        with open(testfile, 'w') as f:
            f.write("hello world\n")
            f.flush()

            self.wait_until(
                lambda: self.output_has(1),
                max_timeout=60, poll_interval=1)

        lines_written = 0

        for n in range(3):
            with open(testfile, 'a') as f:

                for i in range(0, 20 + n):
                    f.write("hello world " + str(i) + " " + str(n) + "\n")
                    lines_written = lines_written + 1

                f.flush()

                self.wait_until(
                    lambda: self.output_has(lines_written + 1),
                    max_timeout=60, poll_interval=1)

        filebeat.check_kill_and_wait()

        # Check that output file has the same number of lines as the log file
        output = self.read_output()
        assert len(output) == (3 * 20 + sum(range(0, 3)) + 1)

    def test_new_line_on_open_file(self):
        """
        Checks that filebeat follows future writes to the same
        file. Same as the test_new_line_on_existing_file but this
        time keep the file open and just flush it.
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
        )
        os.mkdir(self.working_dir + "/log/")

        testfile = self.working_dir + "/log/test.log"
        with open(testfile, 'w') as f:
            f.write("hello world\n")
            f.flush()

            filebeat = self.start_beat()

            self.wait_until(
                lambda: self.output_has(lines=1),
                max_timeout=15)

            # now write another line
            f.write("hello world 1\n")
            f.write("hello world 2\n")
            f.flush()

            self.wait_until(
                lambda: self.output_has(lines=3),
                max_timeout=15)

        filebeat.check_kill_and_wait()

        # Check that output file has the same number of lines as the log file
        output = self.read_output()
        assert len(output) == 3

    def test_tail_files(self):
        """
        Tests that every new file discovered is started
        at the end and not beginning
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            tail_files="true",
        )
        os.mkdir(self.working_dir + "/log/")

        testfile = self.working_dir + "/log/test.log"
        with open(testfile, 'w') as f:
            # Write lines before registrar started
            f.write("hello world 1\n")
            f.write("hello world 2\n")
            f.flush()

        # Sleep 1 second to make sure the file is persisted on disk and
        # timestamp is in the past
        time.sleep(1)

        filebeat = self.start_beat()
        self.wait_until(
            lambda: self.log_contains(
                "Start next scan"),
            max_timeout=10)

        with open(testfile, 'a') as f:
            # write additional lines
            f.write("hello world 3\n")
            f.write("hello world 4\n")
            f.flush()

        self.wait_until(
            lambda: self.output_has(lines=2),
            max_timeout=15)

        filebeat.check_kill_and_wait()

        # Make sure output has only 2 and not 4 lines, means it started at
        # the end
        output = self.read_output()
        assert len(output) == 2

    def test_utf8(self):
        """
        Tests that UTF-8 chars don't break our log tailing.
        """

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
            encoding="utf-8",
        )
        os.mkdir(self.working_dir + "/log/")

        testfile = self.working_dir + "/log/test.log"

        filebeat = self.start_beat()
        self.wait_until(
            lambda: self.log_contains(
                "Start next scan"),
            max_timeout=15)

        # Add utf-8 Chars for the first time
        with codecs.open(testfile, "w", "utf_8") as f:
            # Write lines before registrar started

            # Special encoding needed?!?
            f.write("ニコラスRuflin\n")
            f.flush()

            self.wait_until(
                lambda: self.output_has(lines=1), max_timeout=10)

        # Append utf-8 chars to check if it keeps reading
        with codecs.open(testfile, "a", "utf_8") as f:
            # write additional lines
            f.write("Hello\n")
            f.write("薩科Ruflin" + "\n")
            f.flush()

            self.wait_until(
                lambda: self.output_has(lines=1 + 2), max_timeout=10)

        filebeat.check_kill_and_wait()

        # Make sure output has 3
        output = self.read_output()
        assert len(output) == 3

    def test_encodings(self):
        """
        Tests that several common encodings work.
        """

        # Sample texts are from http://www.columbia.edu/~kermit/utf8.html
        encodings = [
            # golang, python, sample text
            ("plain", "ascii", "I can eat glass"),
            ("utf-8", "utf_8",
             "ὕαλον ϕαγεῖν δύναμαι· τοῦτο οὔ με βλάπτει."),
            ("utf-16be", "utf_16_be",
             "Pot să mănânc sticlă și ea nu mă rănește."),
            ("utf-16le", "utf_16_le",
             "काचं शक्नोम्यत्तुम् । नोपहिनस्ति माम् ॥"),
            ("latin1", "latin1",
             "I kå Glas frässa, ond des macht mr nix!"),
            ("BIG5", "big5", "我能吞下玻璃而不傷身體。"),
            ("gb18030", "gb18030", "我能吞下玻璃而不傷身。體"),
            ("euc-kr", "euckr", " 나는 유리를 먹을 수 있어요. 그래도 아프지 않아요"),
            ("euc-jp", "eucjp", "私はガラスを食べられます。それは私を傷つけません。")
        ]

        # create a file in each encoding
        os.mkdir(self.working_dir + "/log/")
        for _, enc_py, text in encodings:
            with codecs.open(self.working_dir + "/log/test-{}".format(enc_py),
                             "w", enc_py) as f:
                f.write(text + "\n")
                f.close()

        # create the config file
        inputs = []
        for enc_go, enc_py, _ in encodings:
            inputs.append({
                "type": "log",
                "allow_deprecated_use": True,
                "path": self.working_dir + "/log/test-{}".format(enc_py),
                "encoding": enc_go
            })
        self.render_config_template(
            template_name="filebeat_inputs",
            inputs=inputs
        )

        # run filebeat
        filebeat = self.start_beat()
        self.wait_until(lambda: self.output_has(lines=len(encodings)),
                        max_timeout=25)

        # write another line in all files
        for _, enc_py, text in encodings:
            with codecs.open(self.working_dir + "/log/test-{}".format(enc_py),
                             "a", enc_py) as f:
                f.write(text + " 2" + "\n")
                f.close()

        # wait again
        self.wait_until(lambda: self.output_has(lines=len(encodings) * 2),
                        max_timeout=60)
        filebeat.check_kill_and_wait()

        # check that all outputs are present in the JSONs in UTF-8
        # encoding
        output = self.read_output()
        lines = [o["message"] for o in output]
        for _, _, text in encodings:
            assert text in lines
            assert text + " 2" in lines

    def test_file_no_permission(self):
        """
        Checks that filebeat handles files without reading permission well
        """
        if os.name == "nt":
            # Currently skipping this test on windows as it requires `pip install win32api`
            # which seems to have windows only dependencies.
            # To solve this problem a requirements_windows.txt could be introduced which would
            # then only be used on Windows.
            #
            # Below is some additional code to give some indication on how the implementation
            # to remove permissions on Windows (where os.chmod isn't enough) could look like:
            #
            # from win32 import win32api
            # import win32security
            # import ntsecuritycon as con

            # user, domain, type = win32security.LookupAccountName(
            #     "", win32api.GetUserName())
            # sd = win32security.GetFileSecurity(
            #     testfile, win32security.DACL_SECURITY_INFORMATION)

            # dacl = win32security.ACL()
            # # Remove all access rights
            # dacl.AddAccessAllowedAce(win32security.ACL_REVISION, 0, user)

            # sd.SetSecurityDescriptorDacl(1, dacl, 0)
            # win32security.SetFileSecurity(
            #     testfile, win32security.DACL_SECURITY_INFORMATION, sd)
            raise unittest.SkipTest("Requires win32api be installed")
        if os.name != "nt" and os.geteuid() == 0:
            # root ignores permission flags, so we have to skip the test
            raise unittest.SkipTest

        self.render_config_template(
            path=os.path.abspath(self.working_dir) + "/log/*",
        )
        os.mkdir(self.working_dir + "/log/")

        testfile = self.working_dir + "/log/test.log"
        file = open(testfile, 'w')

        iterations = 3
        for n in range(0, iterations):
            file.write("Hello World" + str(n))
            file.write("\n")

        file.close()

        # Remove reading rights from file. On Windows this can only set the read-only flag:
        # https://docs.python.org/3/library/os.html#os.chmod
        os.chmod(testfile, 0o000)

        filebeat = self.start_beat()

        self.wait_until(
            lambda: self.log_contains("permission denied"),
            max_timeout=15)

        filebeat.check_kill_and_wait()

        os.chmod(testfile, 0o755)

        assert False == os.path.isfile(
            os.path.join(self.working_dir, "output/filebeat"))
