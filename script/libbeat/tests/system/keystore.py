from base import BaseTest
import os
from beat.beat import Proc


class KeystoreBase(BaseTest):
    """
    KeystoreBase provides a simple way to add secrets to an existing store
    """

    def add_secret(self, key, value="hello world\n", force=False):
        """
        Add new secret using the --stdin option
        """
        args = [self.test_binary,
                "-systemTest",
                "-test.coverprofile",
                os.path.join(self.working_dir, "coverage.cov"),
                "-c", os.path.join(self.working_dir, "mockbeat.yml"),
                "-e", "-v", "-d", "*",
                "keystore", "add", key, "--stdin",
                ]

        if force:
            args.append("--force")

        proc = Proc(args, os.path.join(self.working_dir, "mockbeat.log"))

        os.write(proc.stdin_write, value)
        os.close(proc.stdin_write)

        return proc.start().wait()
