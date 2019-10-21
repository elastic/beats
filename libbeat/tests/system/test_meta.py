from base import BaseTest

import os
import stat
import unittest
from beat.beat import INTEGRATION_TESTS


class TestMetaFile(BaseTest):
    """
    Test meta file
    """

    def setUp(self):
        super(BaseTest, self).setUp()

        self.meta_file_path = os.path.join(self.working_dir, "data", "meta.json")

        self.render_config_template()
        proc = self.start_beat()
        self.wait_until(lambda: self.log_contains("mockbeat start running."))
        proc.check_kill_and_wait()

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    def test_is_created(self):
        """
        Test that the meta file is created
        """
        self.assertTrue(os.path.exists(self.meta_file_path))

    @unittest.skipUnless(INTEGRATION_TESTS, "integration test")
    def test_has_correct_perms(self):
        """
        Test that the meta file has correct permissions
        """
        perms = oct(stat.S_IMODE(os.lstat(self.meta_file_path).st_mode))
        self.assertEqual(perms, "0600")
