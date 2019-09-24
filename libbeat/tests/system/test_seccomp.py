import platform
import unittest
from base import BaseTest


def is_version_below(version, target):
    t = map(int, target.split('.'))
    v = map(int, version.split('.'))
    v += [0] * (len(t) - len(v))
    for i in range(len(t)):
        if v[i] != t[i]:
            return v[i] < t[i]
    return False


# Require Linux greater or equal than 3.17 and 386/amd64 platform
def is_seccomp_supported():
    p = platform.platform().split('-')
    if p[0] != 'Linux':
        return False
    if is_version_below(p[1], '3.17'):
        return False
    return {'i386', 'i686', 'x86_64', 'amd64'}.intersection(p)


@unittest.skipUnless(is_seccomp_supported(), "Requires Linux 3.17 or greater and i386/amd64 architecture")
class Test(BaseTest):
    """
    Test Beat seccomp policy is loaded
    """

    def setUp(self):
        super(BaseTest, self).setUp()

    def test_seccomp_installed(self):
        """
        Test seccomp policy is installed
        """
        self.render_config_template(
        )
        proc = self.start_beat(extra_args=["-N"])
        self.wait_until(lambda: self.log_contains("Syscall filter successfully installed"))

        proc.kill_and_wait()
