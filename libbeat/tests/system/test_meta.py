from base import BaseTest
from elasticsearch import Elasticsearch, TransportError

import logging
import os
import stat


INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class TestMetaFile(BaseTest):
    """
    Test meta file
    """

    def setUp(self):
        super(BaseTest, self).setUp()

        self.meta_file_path = os.path.join(self.working_dir, "data", "meta.json")

        self.render_config_template()
        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("Beat metadata path: " + self.meta_file_path))
        proc.check_kill_and_wait()

    def test_is_created(self):
        """
        Test that the meta file is created
        """
        self.assertTrue(os.path.exists(self.meta_file_path))

    def test_has_correct_perms(self):
        """
        Test that the meta file has correct permissions
        """
        perms = oct(stat.S_IMODE(os.lstat(self.meta_file_path).st_mode))
        self.assertEqual(perms, "0600")
