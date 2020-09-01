import jinja2
import os
import sys
from beat import common_tests
from filebeat import BaseTest as FilebeatTest


class FilebeatXPackTest(FilebeatTest, common_tests.TestExportsMixin):

    @classmethod
    def setUpClass(self):
        self.beat_name = "filebeat"
        self.beat_path = os.path.abspath(
            os.path.join(os.path.dirname(__file__), "../../"))

        super(FilebeatTest, self).setUpClass()

    def setUp(self):
        super(FilebeatTest, self).setUp()

        # Hack to make jinja2 have the right paths
        self.template_env = jinja2.Environment(
            loader=jinja2.FileSystemLoader([
                os.path.abspath(os.path.join(self.beat_path, "../../filebeat")),
                os.path.abspath(os.path.join(self.beat_path, "../../libbeat"))
            ])
        )
