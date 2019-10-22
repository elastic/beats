from base import BaseTest

import os
import stat
import unittest
import sys

INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)


class TestUmask(BaseTest):
    """
    Test default umask
    """

    DEFAULT_UMASK = 0027

    def setUp(self):
        super(BaseTest, self).setUp()

        self.output_file_permissions = 0666

        self.render_config_template(output_file_permissions=self.output_file_permissions)
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=2)
        proc.check_kill_and_wait()

    @unittest.skipIf(sys.platform.startswith("win"), "umask is not available on Windows")
    def test_output_file_perms(self):
        """
        Test that output file permissions respect default umask
        """
        output_file_path = os.path.join(self.working_dir, "output", "mockbeat")
        perms = stat.S_IMODE(os.lstat(output_file_path).st_mode)

        self.assertEqual(perms, self.output_file_permissions & ~TestUmask.DEFAULT_UMASK)
