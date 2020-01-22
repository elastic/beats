from base import BaseTest


class TestCommandCompletion(BaseTest):
    """
    Test beat completion subcommand
    """

    def setUp(self):
        super(BaseTest, self).setUp()

    def test_bash_completion(self):
        exit_code = self.run_beat(extra_args=["completion", "bash"])
        assert exit_code == 0
        assert self.log_contains("bash completion for mockbeat")

    def test_zsh_completion(self):
        exit_code = self.run_beat(extra_args=["completion", "zsh"])
        assert exit_code == 0
        assert self.log_contains("#compdef mockbeat")

    def test_unknown_completion(self):
        exit_code = self.run_beat(extra_args=["completion", "awesomeshell"])
        assert exit_code == 1
        assert self.log_contains("Unknown shell awesomeshell")
