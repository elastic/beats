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
