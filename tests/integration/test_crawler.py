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
        time.sleep(10)
        proc.kill_and_wait()

        i = 0

        # Count lines of filebeat generated file
        with open(self.working_dir + "/output/filebeat") as f:
            for i, l in enumerate(f, 1):
                pass


        assert iterations == i




