package kubernetes

import (
	"io/ioutil"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/elastic/beats/libbeat/logp"
)

const defaultNode = "localhost"

// GetKubernetesClientset returns a kubernetes clientset. If inCluster is true, it returns an
// in cluster configuration based on the secrets mounted in the Pod. If kubeConfig is passed,
// it parses the config file to get the config required to build a clientset.
func GetKubernetesClientset(inCluster bool, kubeConfig string) (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error
	if inCluster == true {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
		if err != nil {
			return nil, err
		}
	}
	config.ContentType = "application/vnd.kubernetes.protobuf"
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

// DiscoverKubernetesNode figures out the Kubernetes node to use.
// If host is provided in the config use it directly.
// If beat is deployed in k8s cluster, use hostname of pod which is pod name to query pod meta for node name.
// If beat is deployed outside k8s cluster, use machine-id to match against k8s nodes for node name.
func DiscoverKubernetesNode(host string, inCluster bool, clientset *kubernetes.Clientset) (node string) {
	if host != "" {
		logp.Info("kubernetes: Using node %s provided in the config", host)
		return host
	}

	if inCluster {
		ns := inClusterNamespace()
		podName, err := os.Hostname()
		if err != nil {
			logp.Err("kubernetes: Couldn't get hostname as beat pod name in cluster with error: ", err.Error())
			return defaultNode
		}
		logp.Info("kubernetes: Using pod name %s and namespace %s to discover kubernetes node", podName, ns)
		pod, err := clientset.CoreV1().Pods(inClusterNamespace()).Get(podName, v1.GetOptions{})
		if err != nil {
			logp.Err("kubernetes: Querying for pod failed with error: ", err.Error())
			return defaultNode
		}
		logp.Info("kubernetes: Using node %s discovered by in cluster pod node query", pod.Spec.NodeSelector)
		return pod.Spec.NodeName
	}

	mid := machineID()
	if mid == "" {
		logp.Err("kubernetes: Couldn't collect info from any of the files in /etc/machine-id /var/lib/dbus/machine-id")
		return defaultNode
	}

	nodes, err := clientset.CoreV1().Nodes().List(v1.ListOptions{})
	if err != nil {
		logp.Err("kubernetes: Querying for nodes failed with error: ", err.Error())
		return defaultNode
	}
	for _, n := range nodes.Items {
		if n.Status.NodeInfo.MachineID == mid {
			logp.Info("kubernetes: Using node %s discovered by machine-id matching", n.GetName())
			return n.GetName()
		}
	}

	logp.Warn("kubernetes: Couldn't discover node, using localhost as default")
	return defaultNode
}

// inClusterNamespace gets namespace from serviceaccount when beat is in cluster.
// code borrowed from client-go with some changes.
func inClusterNamespace() string {
	// This way assumes you've set the POD_NAMESPACE environment variable using the downward API.
	// This check has to be done first for backwards compatibility with the way InClusterConfig was originally set up
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns
	}

	// Fall back to the namespace associated with the service account token, if available
	if data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}

	return "default"
}

// machineID borrowed from cadvisor.
func machineID() string {
	for _, file := range []string{
		"/etc/machine-id",
		"/var/lib/dbus/machine-id",
	} {
		id, err := ioutil.ReadFile(file)
		if err == nil {
			return strings.TrimSpace(string(id))
		}
	}
	return ""
}
