import contextlib
import os
import shutil
import subprocess
import sys
import yaml


INTEGRATION_TESTS = os.environ.get('INTEGRATION_TESTS', False)

KUBEADM_PATCH = """\
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
readOnlyPort: %d
"""


class KindMixin(object):
    """
    Manage kind to ensure a running Kubernetes cluster is usable during tests.
    """

    # Configuration for the Kind config.
    KIND_CONFIG = {}

    # Timeout waiting for cluster to come up.
    KIND_TIMEOUT = 300

    # Setup kubelet to have its readonly port listening.
    KIND_KUBELET_READONLY_PORT = 10255

    # List extra port mappings.
    KIND_EXTRA_PORT_MAPPINGS = []

    @staticmethod
    def kind_available():
        """
        Check whether `kind` and `kubectl` is useabled.
        """
        return shutil.which("kind") is not None and shutil.which("kubectl") is not None

    @classmethod
    def kind_cluster_name(cls):
        """
        Get the name of the cluster.
        """
        class_dir = os.path.abspath(os.path.dirname(sys.modules[cls.__module__].__file__))
        basename = os.path.basename(class_dir)

        def positivehash(x):
            return hash(x) % ((sys.maxsize+1) * 2)

        return "kind_%s_%X" % (basename, positivehash(frozenset(cls.KIND_CONFIG.items())))

    @classmethod
    def kind_config(cls):
        """
        Return the configuration for kind.
        """
        cfg = {
            "kind": "Cluster",
            "apiVersion": "kind.x-k8s.io/v1alpha4",
            "kubeadmConfigPatches": [KUBEADM_PATCH % cls.KIND_KUBELET_READONLY_PORT],
            "nodes": [
                {
                    "role": "control-plane",
                    "extraPortMappings": [
                        {
                            "containerPort": cls.KIND_KUBELET_READONLY_PORT,
                            "hostPort": cls.KIND_KUBELET_READONLY_PORT,
                        },
                    ] + cls.KIND_EXTRA_PORT_MAPPINGS,
                },
            ],
        }
        cfg.update(cls.KIND_CONFIG)
        return yaml.dump(cfg)

    @classmethod
    def kind_kubecfg_path(cls, build_path):
        """
        Path to kubecfg for the kind cluster.
        """
        cluster_name = cls.kind_cluster_name()
        workdir = os.path.join(build_path, cluster_name)
        return os.path.join(workdir, "kubecfg")

    @classmethod
    def kind_kubectl(cls, build_path, args, capture_output=False, check=True, input=None):
        """
        Execute kubectl against the kind cluster.
        """
        if input is not None:
            input = input.encode("utf-8")
        kubecfg_path = cls.kind_kubecfg_path(build_path)
        args = ["kubectl", "--kubeconfig", kubecfg_path] + args
        return subprocess.run(args, capture_output=capture_output, check=check, input=input)

    @classmethod
    @contextlib.contextmanager
    def kind_kubectl_with_manifest(cls, build_path, manifest):
        """
        Runs with the manifest applied then deletes it.
        """
        cls.kind_kubectl(cls.build_path, ["apply", "-f", "-"], input=manifest)
        try:
            yield
        finally:
            cls.kind_kubectl(cls.build_path, ["delete", "-f", "-"], input=manifest)

    @classmethod
    def kind_kubelet_hosts(cls):
        """
        Kubelet host connection.
        """
        return ["localhost:%d" % cls.KIND_KUBELET_READONLY_PORT]

    @classmethod
    def kind_create_cluster(cls, build_path):
        """
        Create the kind cluster.
        """
        if not INTEGRATION_TESTS:
            return
        if not cls.kind_available():
            return

        cluster_name = cls.kind_cluster_name()
        workdir = os.path.join(build_path, cluster_name)
        kindcfg = os.path.join(workdir, "kindcfg")
        kubecfg = cls.kind_kubecfg_path(build_path)

        if not os.path.exists(workdir):
            os.mkdir(workdir)
        with open(kindcfg, "w") as fp:
            fp.write(cls.kind_config())

        try:
            subprocess.run([
                "kind", "create", "cluster",
                "--name", cluster_name,
                "--config", kindcfg,
                "--kubeconfig", kubecfg,
                "--wait", "%ds" % cls.KIND_TIMEOUT,
            ], check=True)
        except subprocess.CalledProcessError as exc:
            raise Exception("Failed to bring up kind cluster.") from exc

    @classmethod
    def kind_delete_cluster(cls, build_path):
        """
        Delete the kind cluster.
        """
        if not INTEGRATION_TESTS:
            return
        if not cls.kind_available():
            return

        cluster_name = cls.kind_cluster_name()
        kubecfg = cls.kind_kubecfg_path(build_path)

        try:
            subprocess.run([
                "kind", "delete", "cluster",
                "--name", cluster_name,
                "--kubeconfig", kubecfg,
            ], check=True)
        except subprocess.CalledProcessError as exc:
            raise Exception("Failed to bring down kind cluster.") from exc
