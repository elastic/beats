from beat.beat import TestCase


class BaseTest(TestCase):

    @classmethod
    def setUpClass(self):
        self.beat_name = "mockbeat"
        self.build_path = "../../build/system-tests/"
        self.beat_path = "../../libbeat.test"

    def test_version(self):
        """
        Tests -version prints a version and exits.
        """
        self.start_beat(extra_args=["-version"]).check_wait()
        assert self.log_contains("beat version") is True
