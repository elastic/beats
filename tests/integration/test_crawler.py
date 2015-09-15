from filebeat import TestCase

import os
import time


class Test(TestCase):
    def test_crawler(self):
        self.render_config_template(
            #path="/var/log/*"
            path=os.path.abspath(self.working_dir) + "/log/*"
        )
        os.mkdir(self.working_dir + "/log/")

        testfile =  self.working_dir + "/log/test.log"
        file = open(testfile, 'w')

        iterations = 80
        for n in range(0, iterations):
            file.write("hello world" + str(n))
            file.write("\n")


        file.close()

        proc = self.start_filebeat()

        # TODO: Find better solution when filebeat did crawl the file
        time.sleep(30)
        proc.kill_and_wait()

        i = 0

        # Count lines of filebeat generated file
        with open(self.working_dir + "/output/filebeat") as f:
            for i, l in enumerate(f, 1):
                pass

        # Check that output file has the same number of lines as the log file
        assert iterations == i



    def test_unfinished_line(self):
        self.render_config_template(
            #path="/var/log/*"
            path=os.path.abspath(self.working_dir) + "/log/*"
        )
        os.mkdir(self.working_dir + "/log/")

        testfile =  self.working_dir + "/log/test.log"
        file = open(testfile, 'w')

        iterations = 80
        for n in range(0, iterations):
            file.write("hello world" + str(n))
            file.write("\n")

        # An additional line is written to the log file. This line should not be read
        # as there is no finishing \n or \r
        file.write("unfinished line")


        file.close()

        proc = self.start_filebeat()

        # TODO: Find better solution when filebeat did crawl the file
        time.sleep(30)
        response = proc.kill_and_wait()

        print response

        i = 0

        # Count lines of filebeat generated file
        with open(self.working_dir + "/output/filebeat") as f:
            for i, l in enumerate(f, 1):
                pass

        # Check that output file has the same number of lines as the log file
        assert iterations == i


