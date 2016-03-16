import os
import sys

# Load parent directory
sys.path.append(os.path.dirname(os.path.dirname(__file__)))

from metricbeat.metricbeat import BaseTest

class Test(BaseTest):

    def test_base(self):
        """
        Basic test with exiting metricbeat with redis info metricset normally
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

    def test_selectors(self):
        """
        Test if selectors reduce the output as expected
        """
        self.render_config_template(
            redis=True,
            redis_host=os.getenv('REDIS_HOST'),
            redis_selectors=["clients", "cpu"]
        )

        proc = self.start_beat()
        self.wait_until(
            lambda: self.output_has(lines=1)
        )

        output = self.read_output_json()
        event = output[0]
        redis_info = event["redis-info"]

        assert len(redis_info) == 2
        assert len(redis_info["clients"]) == 4
        assert len(redis_info["cpu"]) == 4


        exit_code = proc.kill_and_wait()
        assert exit_code == 0
