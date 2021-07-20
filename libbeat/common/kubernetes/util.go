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

var namespaceFilePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
var osHostname = os.Hostname
var machineId = machineID

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
// If beat is deployed in k8s cluster, use hostname of pod which is pod name to query pod meta for node name.
// If beat is deployed outside k8s cluster, use machine-id to match against k8s nodes for node name.
// If node cannot be discovered, return NODE_NAME env var as default value. In case it is not set return error.
func DiscoverKubernetesNode(log *logp.Logger, host string, inCluster bool, client kubernetes.Interface) (string, error) {
	if host != "" {
		log.Infof("kubernetes: Using node %s provided in the config", host)
		return host, nil
	}
	nodeNameEnv := os.Getenv("NODE_NAME")
	var envError, logerror error
	if nodeNameEnv == "" {
		envError = errors.New("kubernetes: NODE_NAME environment variable was not set")
	}
	ctx := context.TODO()
	if inCluster {
		ns, err := InClusterNamespace()
		if err != nil {
			logerror = fmt.Errorf("kubernetes: Couldn't get namespace when beat is in cluster with error: %+v", err.Error())
			log.Error(logerror)
			return nodeNameEnv, errors.Wrap(envError, logerror.Error())
		}
		podName, err := osHostname()
		if err != nil {
			logerror = fmt.Errorf("kubernetes: Couldn't get hostname as beat pod name in cluster with error: %+v", err.Error())
			log.Error(logerror)
			return nodeNameEnv, errors.Wrap(envError, logerror.Error())
		}
		log.Infof("kubernetes: Using pod name %s and namespace %s to discover kubernetes node", podName, ns)
		pod, err := client.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			logerror = fmt.Errorf("kubernetes: Querying for pod failed with error: %+v", err)
			log.Error(logerror)
			return nodeNameEnv, errors.Wrap(envError, logerror.Error())
		}
		log.Infof("kubernetes: Using node %s discovered by in cluster pod node query", pod.Spec.NodeName)
		return pod.Spec.NodeName, nil
	}

	mid := machineId()
	if mid == "" {
		logerror = errors.New("kubernetes: Couldn't collect info from any of the files in /etc/machine-id /var/lib/dbus/machine-id")
		log.Error(logerror)
		return nodeNameEnv, errors.Wrap(envError, logerror.Error())
	}

	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		logerror = fmt.Errorf("kubernetes: Querying for nodes failed with error: %+v", err)
		log.Error(logerror)
		return nodeNameEnv, errors.Wrap(envError, logerror.Error())
	}
	for _, n := range nodes.Items {
		if n.Status.NodeInfo.MachineID == mid {
			name := n.GetObjectMeta().GetName()
			log.Infof("kubernetes: Using node %s discovered by machine-id matching", name)
			return name, nil
		}
	}

	log.Warn("kubernetes: Couldn't discover node, returning default")
	return nodeNameEnv, envError
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

// InClusterNamespace gets namespace from serviceaccount when beat is in cluster.
// code borrowed from client-go with some changes.
func InClusterNamespace() (string, error) {
	// get namespace associated with the service account token, if available
	data, err := ioutil.ReadFile(namespaceFilePath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
