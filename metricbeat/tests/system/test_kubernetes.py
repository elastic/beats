import os
import metricbeat
import unittest

KUBERNETES_FIELDS = metricbeat.COMMON_FIELDS + ["kubernetes"]


class Test(metricbeat.BaseTest):

    # Tests are disabled as current docker-compose settings fail to start in many cases:
    # COMPOSE_SERVICES = ['kubernetes']  # 'kubestate']

    @unittest.skipUnless(False and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_kubelet_node(self):
        """ Kubernetes kubelet node metricset tests """
        self._test_metricset('node', 1, self.get_kubelet_hosts())

    @unittest.skipUnless(False and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_kubelet_system(self):
        """ Kubernetes kubelet system metricset tests """
        self._test_metricset('system', 2, self.get_kubelet_hosts())

    @unittest.skipUnless(False and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_kubelet_pod(self):
        """ Kubernetes kubelet pod metricset tests """
        self._test_metricset('pod', 1, self.get_kubelet_hosts())

    @unittest.skipUnless(False and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_kubelet_container(self):
        """ Kubernetes kubelet container metricset tests """
        self._test_metricset('container', 1, self.get_kubelet_hosts())

    @unittest.skipUnless(metricbeat.INTEGRATION_TESTS, "integration test")
    @unittest.skip("flacky kube-state-metrics container healthcheck")
    def test_state_node(self):
        """ Kubernetes state node metricset tests """
        self._test_metricset('state_node', 1, self.get_kube_state_hosts())

    @unittest.skipUnless(False and metricbeat.INTEGRATION_TESTS, "integration test")
    @unittest.skip("flacky kube-state-metrics container healthcheck")
    def test_state_pod(self):
        """ Kubernetes state pod metricset tests """
        self._test_metricset('state_pod', 1, self.get_kube_state_hosts())

    @unittest.skipUnless(False and metricbeat.INTEGRATION_TESTS, "integration test")
    @unittest.skip("flacky kube-state-metrics container healthcheck")
    def test_state_container(self):
        """ Kubernetes state container metricset tests """
        self._test_metricset('state_container', 1, self.get_kube_state_hosts())

    def _test_metricset(self, metricset, expected_events, hosts):
        self.render_config_template(modules=[{
            "name": "kubernetes",
            "enabled": "true",
            "metricsets": [metricset],
            "hosts": hosts,
            "period": "5s",
            "extras": {
                "add_metadata": "false",
            }
        }])

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0)
        proc.check_kill_and_wait()

        # Ensure no errors or warnings exist in the log.
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertEqual(len(output), expected_events)
        evt = output[0]

        self.assertItemsEqual(self.de_dot(KUBERNETES_FIELDS), evt.keys(), evt)

        self.assert_fields_are_documented(evt)

    @classmethod
    def get_kubelet_hosts(cls):
        return [
            "http://" +
            os.getenv('KUBELET_HOST', 'localhost') + ':' +
            os.getenv('KUBELET_PORT', '10255')
        ]

    @classmethod
    def get_kube_state_hosts(cls):
        return [
            "http://" +
            os.getenv('KUBE_STATE_METRICS_HOST', 'localhost') + ':' +
            os.getenv('KUBE_STATE_METRICS_PORT', '18080')
        ]
