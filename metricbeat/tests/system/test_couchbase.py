import os
import metricbeat
import unittest


class Test(metricbeat.BaseTest):

    COMPOSE_SERVICES = ['couchbase']

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_bucket(self):
        """
        couchbase bucket metricset test
        """
        self.render_config_template(modules=[{
            "name": "couchbase",
            "metricsets": ["bucket"],
            "hosts": self.get_hosts(),
            "period": "1s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=20)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertTrue(len(output) >= 1)
        evt = output[0]
        print evt

        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_cluster(self):
        """
        couchbase cluster metricset test
        """
        self.render_config_template(modules=[{
            "name": "couchbase",
            "metricsets": ["cluster"],
            "hosts": self.get_hosts(),
            "period": "1s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=20)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertTrue(len(output) >= 1)
        evt = output[0]
        print evt

        self.assert_fields_are_documented(evt)

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_node(self):
        """
        couchbase node metricset test
        """
        self.render_config_template(modules=[{
            "name": "couchbase",
            "metricsets": ["node"],
            "hosts": self.get_hosts(),
            "period": "1s"
        }])
        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=20)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertTrue(len(output) >= 1)
        evt = output[0]
        print evt

        self.assert_fields_are_documented(evt)

    def get_hosts(self):
        return [os.getenv('COUCHBASE_HOST', 'localhost') + ':' +
                os.getenv('COUCHBASE_PORT', '8091')]
