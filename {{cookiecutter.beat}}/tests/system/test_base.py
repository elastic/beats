from {{cookiecutter.beat}} import BaseTest

import os


class Test(BaseTest):

    def test_base(self):
        """
        Basic test with exiting {{cookiecutter.beat|capitalize}} normally
        """
        self.render_config_template(
                path=os.path.abspath(self.working_dir) + "/log/*"
        )

        {{cookiecutter.beat|lower}}_proc = self.start_beat()
        self.wait_until( lambda: self.log_contains("{{cookiecutter.beat}} is running"))
        exit_code = {{cookiecutter.beat|lower}}_proc.kill_and_wait()
        assert exit_code == 0
