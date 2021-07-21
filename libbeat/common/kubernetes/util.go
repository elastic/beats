// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package kubernetes

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/logp"
)

const namespaceFilePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

type discoveryUtils struct {
	eDisc       HostDiscovery
	client      kubernetes.Interface
	isInCluster bool
}
type HostDiscovery interface {
	GetNamespace() (string, error)
	GetPodName() (string, error)
	GetMachineID() string
}

type hostDiscovery struct{}

func GetKubeConfigEnvironmentVariable() string {
	envKubeConfig := os.Getenv("KUBECONFIG")
	if _, err := os.Stat(envKubeConfig); !os.IsNotExist(err) {
		return envKubeConfig
	}
	return ""
}

// GetKubernetesClient returns a kubernetes client. If inCluster is true, it returns an
// in cluster configuration based on the secrets mounted in the Pod. If kubeConfig is passed,
// it parses the config file to get the config required to build a client.
func GetKubernetesClient(kubeconfig string) (kubernetes.Interface, error) {
	if kubeconfig == "" {
		kubeconfig = GetKubeConfigEnvironmentVariable()
	}

	cfg, err := BuildConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("unable to build kube config due to error: %+v", err)
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to build kubernetes clientset: %+v", err)
	}

	return client, nil
}

// BuildConfig is a helper function that builds configs from a kubeconfig filepath.
// If kubeconfigPath is not passed in we fallback to inClusterConfig.
// If inClusterConfig fails, we fallback to the default config.
// This is a copy of `clientcmd.BuildConfigFromFlags` of `client-go` but without the annoying
// klog messages that are not possible to be disabled.
func BuildConfig(kubeconfigPath string) (*restclient.Config, error) {
	if kubeconfigPath == "" {
		kubeconfig, err := restclient.InClusterConfig()
		if err == nil {
			return kubeconfig, nil
		}
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: ""}}).ClientConfig()
}

// IsInCluster takes a kubeconfig file path as input and deduces if Beats is running in cluster or not,
// taking into consideration the existence of KUBECONFIG variable
func IsInCluster(kubeconfig string) bool {
	if kubeconfig != "" || GetKubeConfigEnvironmentVariable() != "" {
		return false
	}
	return true
}

// DiscoverKubernetesNode figures out the Kubernetes node to use.
// If host is provided in the config use it directly.
// If it is empty then return discoverKubernetesNode.
func DiscoverKubernetesNode(log *logp.Logger, configHost string, isInCluster bool, client kubernetes.Interface) (string, error) {
	if configHost != "" {
		log.Infof("kubernetes: Using node %s provided in the config", configHost)
		return configHost, nil
	}
	hd := &hostDiscovery{}
	d := &discoveryUtils{eDisc: hd, client: client, isInCluster: isInCluster}

	return d.discoverKubernetesNode(log)
}

// discoverKubernetesNode figures out the Kubernetes node to use.
// If beat is deployed in k8s cluster, use hostname of pod as the pod name to query pod metadata for node name.
// If beat is deployed outside k8s cluster, use machine-id to match against k8s nodes for node name.
// If node cannot be discovered, return NODE_NAME env var as default value. In case it is not set return error.
func (d *discoveryUtils) discoverKubernetesNode(log *logp.Logger) (string, error) {
	nodeNameEnv := os.Getenv("NODE_NAME")
	var envError error
	var errorMsg string
	if nodeNameEnv == "" {
		envError = errors.New("kubernetes: NODE_NAME environment variable was not set")
	}
	ctx := context.TODO()
	if d.isInCluster {
		ns, err := d.eDisc.GetNamespace()
		if err != nil {
			errorMsg = fmt.Sprintf("kubernetes: Couldn't get namespace when beat is in cluster with error: %+v", err.Error())
			return nodeNameEnv, errors.Wrap(envError, errorMsg)
		}
		podName, err := d.eDisc.GetPodName()
		if err != nil {
			errorMsg = fmt.Sprintf("kubernetes: Couldn't get hostname as beat pod name in cluster with error: %+v", err.Error())
			return nodeNameEnv, errors.Wrap(envError, errorMsg)
		}
		log.Infof("kubernetes: Using pod name %s and namespace %s to discover kubernetes node", podName, ns)
		pod, err := d.client.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			errorMsg = fmt.Sprintf("kubernetes: Querying for pod failed with error: %+v", err)
			return nodeNameEnv, errors.Wrap(envError, errorMsg)
		}
		log.Infof("kubernetes: Using node %s discovered by in cluster pod node query", pod.Spec.NodeName)
		return pod.Spec.NodeName, nil
	}

	mid := d.eDisc.GetMachineID()
	if mid == "" {
		errorMsg = "kubernetes: Couldn't collect info from any of the files in /etc/machine-id /var/lib/dbus/machine-id"
		return nodeNameEnv, errors.Wrap(envError, errorMsg)
	}

	nodes, err := d.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		errorMsg = fmt.Sprintf("kubernetes: Querying for nodes failed with error: %+v", err)
		return nodeNameEnv, errors.Wrap(envError, errorMsg)
	}
	for _, n := range nodes.Items {
		if n.Status.NodeInfo.MachineID == mid {
			name := n.GetObjectMeta().GetName()
			log.Infof("kubernetes: Using node %s discovered by machine-id matching", name)
			return name, nil
		}
	}
	errorMsg = fmt.Sprintf("kubernetes: Couldn't discover node %s", mid)
	return nodeNameEnv, errors.Wrap(envError, errorMsg)
}

//GetMachineID returns the machine-id
// borrowed from machineID of cadvisor.
func (hd *hostDiscovery) GetMachineID() string {
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

// GetNamespace gets namespace from serviceaccount when beat is in cluster.
func (hd *hostDiscovery) GetNamespace() (string, error) {
	return InClusterNamespace()
}

// GetPodName returns the hostname of the pod
func (hd *hostDiscovery) GetPodName() (string, error) {
	return os.Hostname()
}

// InClusterNamespace gets namespace from serviceaccount when beat is in cluster.
// code borrowed from client-go with some changes.
func InClusterNamespace() (string, error) {
	// get namespace associated with the service account token, if available
	data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
