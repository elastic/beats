from filebeat import BaseTest
import os

"""
Tests for the named prospector functionality.
"""


class Test(BaseTest):

    def test_named_prospector(self):
        """
        Test named prospector
        """
        self.render_config_template(
            prospector_name="apache",
            named_paths=os.path.abspath(self.working_dir) + "/log/apache.log",
            prospectors=False,
        )

        os.mkdir(self.working_dir + "/log/")

        apache = self.working_dir + "/log/apache.log"

        with open(apache, 'w') as f:
            f.write("Apache\n")

        filebeat = self.start_beat()

        self.wait_until(
            lambda: self.output_has(lines=1), max_timeout=10)

        filebeat.check_kill_and_wait()

        content = self.read_output()

        # Make sure type is set
        assert content[0]["type"] == "apache"

    def test_named_prospector_with_unnamed(self):
        """
        Test named prospector mixed with unnamed prospector
        """
        self.render_config_template(

            path=os.path.abspath(self.working_dir) + "/log/test.log",
            prospector_name="apache",
            named_paths=os.path.abspath(self.working_dir) + "/log/apache.log",
        )

        os.mkdir(self.working_dir + "/log/")

        testfile = self.working_dir + "/log/test.log"
        apache = self.working_dir + "/log/apache.log"

        with open(testfile, 'w') as f:
            f.write("Hello world\n")

        with open(apache, 'w') as f:
            f.write("Apache\n")

        filebeat = self.start_beat()

        self.wait_until(
            lambda: self.output_has(lines=2), max_timeout=10)

        filebeat.check_kill_and_wait()
