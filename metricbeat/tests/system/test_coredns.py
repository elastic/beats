import os
import metricbeat
import unittest

COREDNS_FIELDS = metricbeat.COMMON_FIELDS + ["coredns"]


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['coredns']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_stats(self):
        """
        coredns stats test
        """
        self.render_config_template(modules=[{
            "name": "coredns",
            "metricsets": ["stats"],
            "hosts": self.get_hosts(),
            "period": "5s",
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            self.assert_fields_are_documented(evt)
            self.assertItemsEqual(self.de_dot(COREDNS_FIELDS), evt.keys(), evt)

    def get_hosts(self):
        return ["{}:{}".format(
            os.getenv('COREDNS_HOST', 'localhost'),
            os.getenv('COREDNS_PORT', '9153')
        )]
