import os
import sys
import unittest

sys.path.append(os.path.join(os.path.dirname(__file__), '../../tests/system'))
import metricbeat
from beat.kind import KindMixin


KUBERNETES_FIELDS = metricbeat.COMMON_FIELDS + ["kubernetes"]
KUBERNETES_CONTAINER_FIELDS = metricbeat.COMMON_FIELDS + ["container", "kubernetes"]

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

STS_MANIFEST = """\
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: basic-sts
  labels:
    app: basic-sts
spec:
  serviceName: basic-sts
  replicas: 1
  selector:
    matchLabels:
      app: basic-sts
  template:
    metadata:
      labels:
        app: basic-sts
    spec:
      containers:
      - name: sh
        image: alpine:3
        command: ["sh", "-c", "sleep infinity"]
        volumeMounts:
        - name: mnt
          mountPath: /mnt
  volumeClaimTemplates:
  - metadata:
      name: mnt
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 1Mi 
"""

DEPLOYMENT_MANIFEST = """\
apiVersion: apps/v1
kind: Deployment
metadata:
  name: basic-deployment
  labels:
    app: basic-deployment
spec:
  replicas: 1
  selector:
    matchLabels:
      app: basic-deployment
  template:
    metadata:
      labels:
        app: basic-deployment
    spec:
      containers:
      - name: sh
        image: alpine:3
        command: ["sh", "-c", "sleep infinity"]
"""

CRONJOB_MANIFEST = """\
apiVersion: batch/v1beta1
kind: CronJob
metadata:
  name: basic-cronjob
spec:
  schedule: "* * * * *"
  jobTemplate:
    spec:
      template:
        metadata:
          name: basic-job
        spec:
          containers:
          - name: hello
            image: alpine:3
            command: ["sh", "-c", "echo Hello!"]
          restartPolicy: Never
"""

RESOURCE_QUOTA_MANIFEST = """\
apiVersion: v1
kind: ResourceQuota
metadata:
  name: object-counts
spec:
  hard:
    configmaps: "99"
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
                cls.kind_kubectl(cls.build_path, ["apply", "-f", "-"], input=STS_MANIFEST)
                cls.kind_kubectl(cls.build_path, ["wait", "--timeout=300s",
                                                  "--for=condition=available", "deployment/kube-state-metrics"])
                cls.kind_kubectl(cls.build_path, ["wait", "--timeout=300s",
                                                  "--for=condition=ready", "pod/basic-sts-0"])
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
    def test_kubelet_apiserver(self):
        """ Kubernetes kubelet apiserver metricset tests """
        self._test_metricset('apiserver', KUBERNETES_FIELDS, self.kind_kubelet_hosts())

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_kubelet_container(self):
        """ Kubernetes kubelet container metricset tests """
        self._test_metricset('container', KUBERNETES_FIELDS, self.kind_kubelet_hosts())

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_kubelet_controllermanager(self):
        """ Kubernetes kubelet controllermanager metricset tests """
        self._test_metricset('controllermanager', KUBERNETES_FIELDS, self.kind_kubelet_hosts())

    # TODO: Get working.
    #@unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    #def test_kubelet_event(self):
    #    """ Kubernetes kubelet event metricset tests """
    #    # This test needs to create some events after starting metricbeat, as it will only capture events
    #    # that occur from the point in time that it was started.
    #    self.render_config_template(modules=[{
    #        "name": "kubernetes",
    #        "enabled": "true",
    #        "metricsets": ["event"],
    #        "hosts": self.kind_kubelet_hosts(),
    #        "period": "5s",
    #        "extras": {
    #            "kube_config": self.kind_kubecfg_path(self.build_path),
    #            "add_metadata": "false",
    #        },
    #    }])
    #    proc = self.start_beat()

    #    # Add the cronjob to create events and wait until an event is captured.
    #    with self.kind_kubectl_with_manifest(self.build_path, DEPLOYMENT_MANIFEST):
    #        self.wait_until(lambda: self.output_lines() > 0, max_timeout=30)
    #        proc.check_kill_and_wait()

    #    # Ensure no errors or warnings exist in the log.
    #    self.assert_no_logged_warnings()

    #    output = self.read_output_json()
    #    self.assertTrue(len(output) > 0)
    #    evt = output[0]

    #    self.assertCountEqual(self.de_dot(KUBERNETES_FIELDS), evt.keys(), evt)

    #    self.assert_fields_are_documented(evt)

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_kubelet_node(self):
        """ Kubernetes kubelet node metricset tests """
        self._test_metricset('node', KUBERNETES_FIELDS, self.kind_kubelet_hosts())

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_kubelet_pod(self):
        """ Kubernetes kubelet pod metricset tests """
        self._test_metricset('pod', KUBERNETES_FIELDS, self.kind_kubelet_hosts())

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_kubelet_proxy(self):
        """ Kubernetes kubelet proxy metricset tests """
        self._test_metricset('proxy', KUBERNETES_FIELDS, self.kind_kubelet_hosts())

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_kubelet_scheduler(self):
        """ Kubernetes kubelet scheduler metricset tests """
        self._test_metricset('scheduler', KUBERNETES_FIELDS, self.kind_kubelet_hosts())

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_kubelet_system(self):
        """ Kubernetes kubelet system metricset tests """
        self._test_metricset('system', KUBERNETES_FIELDS, self.kind_kubelet_hosts())

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_kubelet_volume(self):
       """ Kubernetes kubelet volume metricset tests """
       # Default timeout of 10s is too short for notice of volumes.
       self._test_metricset('volume', KUBERNETES_FIELDS, self.kind_kubelet_hosts(), max_timeout=25)

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_state_container(self):
        """ Kubernetes state container metricset tests """
        self._test_metricset('state_container', KUBERNETES_CONTAINER_FIELDS, self.kube_state_metrics_hosts())

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_state_cronjob(self):
        """ Kubernetes state cronjob metricset tests """
        with self.kind_kubectl_with_manifest(self.build_path, CRONJOB_MANIFEST):
            self._test_metricset('state_cronjob', KUBERNETES_FIELDS, self.kube_state_metrics_hosts())

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_state_deployment(self):
        """ Kubernetes state deployment metricset tests """
        self._test_metricset('state_deployment', KUBERNETES_FIELDS, self.kube_state_metrics_hosts())

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_state_node(self):
        """ Kubernetes state node metricset tests """
        self._test_metricset('state_node', KUBERNETES_FIELDS, self.kube_state_metrics_hosts())

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_state_persistentvolume(self):
        """ Kubernetes state persistentvolume metricset tests """
        self._test_metricset('state_persistentvolume', KUBERNETES_FIELDS, self.kube_state_metrics_hosts())

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_state_persistentvolumeclaim(self):
        """ Kubernetes state persistentvolumeclaim metricset tests """
        self._test_metricset('state_persistentvolumeclaim', KUBERNETES_FIELDS, self.kube_state_metrics_hosts())

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_state_pod(self):
        """ Kubernetes state pod metricset tests """
        self._test_metricset('state_pod', KUBERNETES_FIELDS, self.kube_state_metrics_hosts())

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_state_replicaset(self):
        """ Kubernetes state replicaset metricset tests """
        self._test_metricset('state_replicaset', KUBERNETES_FIELDS, self.kube_state_metrics_hosts())

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_state_resourcequota(self):
       """ Kubernetes state resourcequota metricset tests """
       with self.kind_kubectl_with_manifest(self.build_path, RESOURCE_QUOTA_MANIFEST):
           self._test_metricset('state_resourcequota', KUBERNETES_FIELDS, self.kube_state_metrics_hosts())

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_state_service(self):
        """ Kubernetes state service metricset tests """
        self._test_metricset('state_service', KUBERNETES_FIELDS, self.kube_state_metrics_hosts())

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_state_statefulset(self):
        """ Kubernetes state statefulset metricset tests """
        self._test_metricset('state_statefulset', KUBERNETES_FIELDS, self.kube_state_metrics_hosts())

    @unittest.skipUnless(KindMixin.kind_available() and metricbeat.INTEGRATION_TESTS, "integration test")
    def test_state_storageclass(self):
        """ Kubernetes state storageclass metricset tests """
        self._test_metricset('state_storageclass', KUBERNETES_FIELDS, self.kube_state_metrics_hosts())

    def _test_metricset(self, metricset, fields, hosts, max_timeout=10):
        self.render_config_template(modules=[{
            "name": "kubernetes",
            "enabled": "true",
            "metricsets": [metricset],
            "hosts": hosts,
            "period": "5s",
            "extras": {
                "add_metadata": "false",
            },
        }])

        proc = self.start_beat()
        self.wait_until(lambda: self.output_lines() > 0, max_timeout=max_timeout)
        proc.check_kill_and_wait()

        # Ensure no errors or warnings exist in the log.
        self.assert_no_logged_warnings()

        output = self.read_output_json()
        self.assertTrue(len(output) > 0)
        evt = output[0]

        self.assertCountEqual(self.de_dot(fields), evt.keys(), evt)

        self.assert_fields_are_documented(evt)
