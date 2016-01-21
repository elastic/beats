from {{cookiecutter.beat}} import BaseTest

import os


class Test(BaseTest):

    def test_base(self):
        """
        Basic test with exiting Modbeat normally
        """
        self.render_config_template(
                path=os.path.abspath(self.working_dir) + "/log/*"
        )

        exit_code = self.run_beat()
        assert exit_code == 0
