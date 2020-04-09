import os
import sys
import unittest

sys.path.append(os.path.join(os.path.dirname(__file__), '../../tests/system'))
import metricbeat
from beat.kind import KindMixin


KUBERNETES_FIELDS = metricbeat.COMMON_FIELDS + ["kubernetes"]

KUBE_STATE_METRICS_MANIFEST = """\
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kube-state-metrics
rules:
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["*"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kube-state-metrics
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: kube-state-metrics
subjects:
- kind: ServiceAccount
  name: kube-state-metrics
  namespace: default
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kube-state-metrics
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kube-state-metrics
  labels:
    app: kube-state-metrics
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kube-state-metrics
  template:
    metadata:
      labels:
        app: kube-state-metrics
    spec:
      containers:
      - name: kube-state-metrics
        image: quay.io/coreos/kube-state-metrics:v1.8.0
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 5
          timeoutSeconds: 5
        ports:
        - containerPort: 8080
          name: http-metrics
        - containerPort: 8081
          name: telemetry
        readinessProbe:
          httpGet:
            path: /
            port: 8081
          initialDelaySeconds: 5
          timeoutSeconds: 5
      serviceAccountName: kube-state-metrics
---
apiVersion: v1
kind: Service
metadata:
  name: kube-state-metrics
spec:
  type: NodePort
  selector:
    app: kube-state-metrics
  ports:
    - port: 8080
      targetPort: 8080
      nodePort: 30808
"""


class Test(metricbeat.BaseTest, KindMixin):

    # Map kube-state-metrics port from inside of kind to host.
    KIND_EXTRA_PORT_MAPPINGS = [
        {
            "containerPort": 30808,
            "hostPort": 30808,
        }
    ]

    @classmethod
    def setUpClass(cls):
        super().setUpClass()
        cls.kind_create_cluster(cls.build_path)
        if metricbeat.INTEGRATION_TESTS and cls.kind_available():
            try:
                cls.kind_kubectl(cls.build_path, ["apply", "-f", "-"], input=KUBE_STATE_METRICS_MANIFEST)
                cls.kind_kubectl(cls.build_path, ["wait", "--timeout=300s", "--for=condition=available", "deployment/kube-state-metrics"])
            except:
                cls.kind_delete_cluster(cls.build_path)
                raise

    @classmethod
    def tearDownClass(cls):
        super().tearDownClass()
        cls.kind_delete_cluster(cls.build_path)

    @classmethod
    def kube_state_metrics_hosts(cls):
        """
        kube-state-metrics host connection.
        """
        return ["localhost:30808"]

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_kubelet_node(self):
        """ Kubernetes kubelet node metricset tests """
        self._test_metricset('node', self.kind_kubelet_hosts())

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_kubelet_system(self):
        """ Kubernetes kubelet system metricset tests """
        self._test_metricset('system', self.kind_kubelet_hosts())

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_kubelet_pod(self):
        """ Kubernetes kubelet pod metricset tests """
        self._test_metricset('pod', self.kind_kubelet_hosts())

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_kubelet_container(self):
        """ Kubernetes kubelet container metricset tests """
        self._test_metricset('container', self.kind_kubelet_hosts())

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_state_node(self):
        """ Kubernetes state node metricset tests """
        self._test_metricset('state_node', self.kube_state_metrics_hosts())

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_state_pod(self):
        """ Kubernetes state pod metricset tests """
        self._test_metricset('state_pod', self.kube_state_metrics_hosts())

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_state_container(self):
        """ Kubernetes state container metricset tests """
        self._test_metricset('state_container', self.kube_state_metrics_hosts())

    def _test_metricset(self, metricset, hosts):
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
        self.assertTrue(len(output) > 0)
        evt = output[0]

        self.assertCountEqual(self.de_dot(KUBERNETES_FIELDS), evt.keys(), evt)

        self.assert_fields_are_documented(evt)

    @classmethod
    def get_kubelet_hosts(cls):
        return [self.compose_host("kubernetes")]

    @classmethod
    def get_kube_state_hosts(cls):
        return [self.compose_host("kubestate")]
