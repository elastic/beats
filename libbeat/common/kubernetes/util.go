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

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/safemapstr"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/logp"
)

const namespaceFilePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

type HostDiscoveryUtils interface {
	GetNamespace() (string, error)
	GetPodName() (string, error)
	GetMachineID() string
}

// DiscoverKubernetesNodeParams includes parameters for discovering kubernetes node
type DiscoverKubernetesNodeParams struct {
	ConfigHost  string
	Client      kubernetes.Interface
	IsInCluster bool
	HostUtils   HostDiscoveryUtils
}

// DefaultDiscoveryUtils implements functions of HostDiscoveryUtils interface
type DefaultDiscoveryUtils struct{}

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
func GetKubernetesClient(kubeconfig string, opt KubeClientOptions) (kubernetes.Interface, error) {
	if kubeconfig == "" {
		kubeconfig = GetKubeConfigEnvironmentVariable()
	}

	cfg, err := BuildConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("unable to build kube config due to error: %+v", err)
	}
	cfg.QPS = opt.QPS
	cfg.Burst = opt.Burst
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
// If it is empty then try
// 1. If beat is deployed in k8s cluster, use hostname of pod as the pod name to query pod metadata for node name.
// 2. If step 1 fails or beat is deployed outside k8s cluster, use machine-id to match against k8s nodes for node name.
// 3. If node cannot be discovered with step 1,2, fallback to NODE_NAME env var as default value. In case it is not set return error.
func DiscoverKubernetesNode(log *logp.Logger, nd *DiscoverKubernetesNodeParams) (string, error) {
	ctx := context.TODO()
	// Discover node by configuration file (NODE) if set
	if nd.ConfigHost != "" {
		log.Infof("kubernetes: Using node %s provided in the config", nd.ConfigHost)
		return nd.ConfigHost, nil
	}
	// Discover node by serviceaccount namespace and pod's hostname in case Beats is running in cluster
	if nd.IsInCluster {
		node, err := discoverInCluster(nd, ctx)
		if err == nil {
			log.Infof("kubernetes: Node %s discovered by in cluster pod node query", node)
			return node, nil
		}
		log.Debug(err)
	}

	// try discover node by machine id
	node, err := discoverByMachineId(nd, ctx)
	if err == nil {
		log.Infof("kubernetes: Node %s discovered by machine-id matching", node)
		return node, nil
	}
	log.Debug(err)

	// fallback to environment variable NODE_NAME
	node = os.Getenv("NODE_NAME")
	if node != "" {
		log.Infof("kubernetes: Node %s discovered by NODE_NAME environment variable", node)
		return node, nil
	}

	return "", errors.New("kubernetes: Node could not be discovered with any known method. Consider setting env var NODE_NAME")
}

func discoverInCluster(nd *DiscoverKubernetesNodeParams, ctx context.Context) (node string, errorMsg error) {
	ns, err := nd.HostUtils.GetNamespace()
	if err != nil {
		errorMsg = fmt.Errorf("kubernetes: Couldn't get namespace when beat is in cluster with error: %+v", err.Error())
		return
	}
	podName, err := nd.HostUtils.GetPodName()
	if err != nil {
		errorMsg = fmt.Errorf("kubernetes: Couldn't get hostname as beat pod name in cluster with error: %+v", err.Error())
		return
	}
	pod, err := nd.Client.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		errorMsg = fmt.Errorf("kubernetes: Querying for pod failed with error: %+v", err)
		return
	}
	return pod.Spec.NodeName, nil
}

func discoverByMachineId(nd *DiscoverKubernetesNodeParams, ctx context.Context) (nodeName string, errorMsg error) {
	mid := nd.HostUtils.GetMachineID()
	if mid == "" {
		errorMsg = errors.New("kubernetes: Couldn't collect info from any of the files in /etc/machine-id /var/lib/dbus/machine-id")
		return
	}

	nodes, err := nd.Client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		errorMsg = fmt.Errorf("kubernetes: Querying for nodes failed with error: %+v", err)
		return
	}
	for _, n := range nodes.Items {
		if n.Status.NodeInfo.MachineID == mid {
			nodeName = n.GetObjectMeta().GetName()
			return nodeName, nil
		}
	}
	errorMsg = fmt.Errorf("kubernetes: Couldn't discover node %s", mid)
	return
}

// GetMachineID returns the machine-idadd_kubernetes_metadata/indexers_test.go
// borrowed from machineID of cadvisor.
func (hd *DefaultDiscoveryUtils) GetMachineID() string {
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
func (hd *DefaultDiscoveryUtils) GetNamespace() (string, error) {
	return InClusterNamespace()
}

// GetPodName returns the hostname of the pod
func (hd *DefaultDiscoveryUtils) GetPodName() (string, error) {
	return os.Hostname()
}

// InClusterNamespace gets namespace from serviceaccount when beat is in cluster. // code borrowed from client-go with some changes.
func InClusterNamespace() (string, error) {
	// get namespace associated with the service account token, if available
	data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

type ContainerInPod struct {
	ID      string
	Runtime string
	Spec    Container
	Status  PodContainerStatus
}

// GetContainersInPod returns all the containers defined in a pod and their statuses.
// It includes init and ephemeral containers.
func GetContainersInPod(pod *Pod) []*ContainerInPod {
	var containers []*ContainerInPod
	for _, c := range pod.Spec.Containers {
		containers = append(containers, &ContainerInPod{Spec: c})
	}
	for _, c := range pod.Spec.InitContainers {
		containers = append(containers, &ContainerInPod{Spec: c})
	}
	for _, c := range pod.Spec.EphemeralContainers {
		c := Container(c.EphemeralContainerCommon)
		containers = append(containers, &ContainerInPod{Spec: c})
	}

	statuses := make(map[string]*PodContainerStatus)
	mapStatuses := func(s []PodContainerStatus) {
		for i := range s {
			statuses[s[i].Name] = &s[i]
		}
	}
	mapStatuses(pod.Status.ContainerStatuses)
	mapStatuses(pod.Status.InitContainerStatuses)
	mapStatuses(pod.Status.EphemeralContainerStatuses)
	for _, c := range containers {
		if s, ok := statuses[c.Spec.Name]; ok {
			c.ID, c.Runtime = ContainerIDWithRuntime(*s)
			c.Status = *s
		}
	}

	return containers
}

// PodAnnotations returns the annotations in a pod
func PodAnnotations(pod *Pod) common.MapStr {
	annotations := common.MapStr{}
	for k, v := range pod.GetObjectMeta().GetAnnotations() {
		safemapstr.Put(annotations, k, v)
	}
	return annotations
}

// PodNamespaceAnnotations returns the annotations of the namespace of the pod
func PodNamespaceAnnotations(pod *Pod, watcher Watcher) common.MapStr {
	if watcher == nil {
		return nil
	}

	rawNs, ok, err := watcher.Store().GetByKey(pod.Namespace)
	if !ok || err != nil {
		return nil
	}

	namespace, ok := rawNs.(*Namespace)
	if !ok {
		return nil
	}

	annotations := common.MapStr{}
	for k, v := range namespace.GetAnnotations() {
		safemapstr.Put(annotations, k, v)
	}
	return annotations
}

// PodTerminating returns true if a pod is marked for deletion or is in a phase beyond running.
func PodTerminating(pod *Pod) bool {
	if pod.GetObjectMeta().GetDeletionTimestamp() != nil {
		return true
	}

	switch pod.Status.Phase {
	case PodRunning, PodPending:
	default:
		return true
	}

	return false
}

// PodTerminated returns true if a pod is terminated, this method considers a
// pod as terminated if none of its containers are running (or going to be running).
func PodTerminated(pod *Pod, containers []*ContainerInPod) bool {
	// Pod is not marked for termination, so it is not terminated.
	if !PodTerminating(pod) {
		return false
	}

	// If any container is running, the pod is not terminated yet.
	for _, container := range containers {
		if container.Status.State.Running != nil {
			return false
		}
	}

	return true
}
