import os
import sys

# Load parent directory
sys.path.append(os.path.dirname(os.path.dirname(__file__)))

from metricbeat.metricbeat import BaseTest

class Test(BaseTest):

    def test_base(self):
        """
        Basic test with exiting Mockbeat normally
        """
        self.render_config_template(
            redis=True,
            redis_host=os.getenv('REDIS_HOST')
        )

        proc = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=1)
        )

        exit_code = proc.kill_and_wait()
        assert exit_code == 0
