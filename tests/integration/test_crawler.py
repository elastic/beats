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

        file = open(self.working_dir + "/log/test.log", 'w')

        file.write("hello world")
        file.write("\n")
        file.write("more")
        file.write("\n")
        file.write("to")
        file.write("\n")
        file.write("say")
        file.write("\n")
        file.close()

        proc = self.start_filebeat()

        time.sleep(10)
        proc.kill_and_wait()




