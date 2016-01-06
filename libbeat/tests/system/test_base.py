from mockbeat import TestCase

import os


# Additional tests to be added:
# * Check what happens when file renamed -> no recrawling should happen
# * Check if file descriptor is "closed" when file disappears
class Test(TestCase):
    def test_base(self):
        """
        Checks if all lines are read from the log file.
        """

        self.render_config_template(
        )
        os.mkdir(self.working_dir + "/log/")

        testfile = self.working_dir + "/log/test.log"
        file = open(testfile, 'w')

        iterations = 80
        for n in range(0, iterations):
            file.write("hello world" + str(n))
            file.write("\n")

        file.close()

        mockbeat = self.run_mockbeat()
