import os
import metricbeat
import unittest
import time

GOLANG_FIELDS = metricbeat.COMMON_FIELDS + ["golang"]


class Test(metricbeat.BaseTest):

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_stats(self):
        """
        prometheus stats test
        """
        self.render_config_template(modules=[{
            "name": "golang",
            "metricsets": ["heap"],
            "hosts": self.get_hosts(),
            "period": "1s"
        }])
        proc = self.start_beat(
            extra_args=[
                "-httpprof",
                os.getenv(
                    'GOLANG_HOST',
                    'localhost') +
                ":" +
                os.getenv(
                    'GOLANG_PORT',
                    '6060')])

        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        evt = output[0]

        self.assertItemsEqual(self.de_dot(GOLANG_FIELDS), evt.keys(), evt)
        assert evt["golang"]["heap"]["allocations"]["total"] > 0

        self.assert_fields_are_documented(evt)

    def get_hosts(self):
        return ["http://" + os.getenv('GOLANG_HOST', 'localhost') + ':' +
                os.getenv('GOLANG_PORT', '6060')]
