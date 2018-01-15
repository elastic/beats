package kubernetes

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ericchiang/k8s"
	"github.com/ghodss/yaml"

	"github.com/elastic/beats/libbeat/logp"
)

// GetKubernetesClient returns a kubernetes client. If inCluster is true, it returns an
// in cluster configuration based on the secrets mounted in the Pod. If kubeConfig is passed,
// it parses the config file to get the config required to build a client.
func GetKubernetesClient(inCluster bool, kubeConfig string) (client *k8s.Client, err error) {
	if inCluster == true {
		client, err = k8s.NewInClusterClient()
		if err != nil {
			return nil, fmt.Errorf("Unable to get in cluster configuration: %v", err)
		}
	} else {
		data, err := ioutil.ReadFile(kubeConfig)
		if err != nil {
			return nil, fmt.Errorf("read kubeconfig: %v", err)
		}

		// Unmarshal YAML into a Kubernetes config object.
		var config k8s.Config
		if err = yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("unmarshal kubeconfig: %v", err)
		}
		client, err = k8s.NewClient(&config)
		if err != nil {
			return nil, err
		}
	}

	return client, nil
}

// DiscoverKubernetesNode figures out the Kubernetes host to use. If host is provided in the config
// use it directly. Else use hostname of the pod which is the Pod ID to query the Pod and get the Node
// name from the specification. Else, return localhost as a default.
func DiscoverKubernetesNode(host string, client *k8s.Client) string {
	ctx := context.Background()
	if host == "" {
		podName := os.Getenv("HOSTNAME")
		logp.Info("Using pod name %s and namespace %s", podName, client.Namespace)
		if podName == "localhost" {
			host = "localhost"
		} else {
			pod, err := client.CoreV1().GetPod(ctx, podName, client.Namespace)
			if err != nil {
				logp.Err("Querying for pod failed with error: ", err.Error())
				logp.Info("Unable to find pod, setting host to localhost")
				host = "localhost"
			} else {
				host = pod.Spec.GetNodeName()
			}

		}
	}

	return host
}
