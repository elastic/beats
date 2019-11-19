from xpack_metricbeat import XPackTest
import metricbeat
import unittest

class ActiveMqTest(XPackTest):
    COMPOSE_SERVICES = ['activemq']

    def get_activemq_module_config(self, metricset):
        return {
            "name": "activemq",
            "metricsets": [metricset],
            "period": "5s",
            "hosts": self.get_hosts(),
            "path": "/api/jolokia/?ignoreErrors=true&canonicalNaming=false",
            "username": "admin",
            "password": "admin"
        }

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    def test_broker_metrics_collected(self):
        self.render_config_template(modules=[self.get_activemq_module_config("broker")])
        proc = self.start_beat(home=self.beat_path)
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertGreater(len(output), 0)

        for evt in output:
            assert "name" in evt["activemq"]["broker"]


class TestRelease5130(ActiveMqTest):
    COMPOSE_ENV = {'ACTIVEMQ_VERSION': '5.13.0'}
