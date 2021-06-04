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
	"github.com/coreos/etcd/client"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/elastic/beats/v7/libbeat/logp"
)

const defaultNode = "localhost"

type clusteInfo struct {
	url string
	name string
}

func getKubeConfigEnvironmentVariable() string {
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
		kubeconfig = getKubeConfigEnvironmentVariable()
	}

	cfg, err := buildConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("unable to build kube config due to error: %+v", err)
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to build kubernetes clientset: %+v", err)
	}
	GetKubernetesClusterIdentifier("", client)
	return client, nil
}

func GetKubernetesClusterIdentifier(kubeconfig string, client kubernetes.Interface) (clusteInfo, error) {
	// try with kubeadm-config
	clusterInfo, err := getClusterInfoFromKubeConfigFile(kubeconfig)
	if err == nil {
		return clusterInfo, nil
	}
	// try with kubeadm-config
	clusterInfo, err = getClusterInfoFromKubeadmConfigMap(client)
	if err == nil {
		return clusterInfo, nil
	}
	// try with GKE metadata
	clusterInfo, err = getClusterInfoFromGKEMetadata()
	if err == nil {
		return clusterInfo, nil
	}
	return clusteInfo{}, fmt.Errorf("unable to retrieve cluster identifiers")
}

func getClusterInfoFromGKEMetadata() (clusteInfo, error) {
		// "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/attributes/kubeconfig
		// "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/attributes/cluster-name
		//req, err := http.NewRequest("GET", url, nil)
		//if err != nil {
		//	result.err = errors.Wrapf(err, "failed to create http request for %v", f.provider)
		//	return
		//}
		//for k, v := range f.headers {
		//	req.Header.Add(k, v)
		//}
		//req = req.WithContext(ctx)
		//
		//rsp, err := client.Do(req)
		//if err != nil {
		//	result.err = errors.Wrapf(err, "failed requesting %v metadata", f.provider)
		//	return
		//}
		//defer rsp.Body.Close()
		//
		//if rsp.StatusCode != http.StatusOK {
		//	result.err = errors.Errorf("failed with http status code %v", rsp.StatusCode)
		//	return
		//}
		//
		//all, err := ioutil.ReadAll(rsp.Body)
		//if err != nil {
		//	result.err = errors.Wrapf(err, "failed requesting %v metadata", f.provider)
		//	return
		//}

		// Decode JSON.
		//err = responseHandler(all, result)
		//if err != nil {
		//	result.err = err
		//	return
		//}

	return clusteInfo{}, fmt.Errorf("unable to get cluster identifiers from GKE metadata")
}

type ClusterConfiguration struct {
	ApiServer ApiServer      `yaml:"apiServer"`
	ClusterName     string `yaml:"clusterName"`
}

type ApiServer struct {
	CertSANs []string `yaml:"certSANs"`
}

type ClusterStatus struct {
	ApiEndpoints map[string]ApiEndpoint      `yaml:"apiEndpoints"`
}

type ApiEndpoint struct {
	AdvertiseAddress string `yaml:"advertiseAddress"`
}

func getClusterInfoFromKubeadmConfigMap(client kubernetes.Interface) (clusteInfo, error) {
	clusterInfo := clusteInfo{}
	cm, err := client.CoreV1().ConfigMaps("kube-system").Get(context.TODO(), "kubeadm-config", metav1.GetOptions{})
	if err != nil {
		return clusteInfo{}, fmt.Errorf("unable to get cluster identifiers from kubeadm-config: %+v", err)
	}
	p := cm.Data["ClusterConfiguration"]

	cc := &ClusterConfiguration{}
	err = yaml.Unmarshal([]byte(p), cc)
	if err != nil {
		return clusteInfo{}, err
	}
	if cc.ClusterName != "" {
		clusterInfo.name = cc.ClusterName
	}

	p = cm.Data["ClusterStatus"]
	cs := &ClusterStatus{}
	err = yaml.Unmarshal([]byte(p), cs)
	if err != nil {
		return clusteInfo{}, err
	}
	if len(cs.ApiEndpoints) > 0 {
		for _, element := range cs.ApiEndpoints {
			if element.AdvertiseAddress != "" {
				clusterInfo.url = element.AdvertiseAddress
			}
		}
	} else if len(cc.ApiServer.CertSANs) > 0 {
		// TODO: verify that the last one is what we need here
		idx := len(cc.ApiServer.CertSANs) - 1
		clusterInfo.url = cc.ApiServer.CertSANs[idx]
	}
	fmt.Println(clusterInfo)
	return clusterInfo, nil
}

func getClusterInfoFromKubeConfigFile(kubeconfig string) (clusteInfo, error) {
	if kubeconfig == "" {
		kubeconfig = getKubeConfigEnvironmentVariable()
	}

	if kubeconfig == "" {
		return clusteInfo{}, fmt.Errorf("unable to build cluster identifiers from kube_config from env")
	}

	cfg, err := buildConfig(kubeconfig)
	if err != nil {
		return clusteInfo{}, fmt.Errorf("unable to build kube config due to error: %+v", err)
	}


	cfg2, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return clusteInfo{}, fmt.Errorf("unable to load kube_config due to error: %+v", err)
	}

	for key, element := range cfg2.Clusters {
		if element.Server == cfg.Host {
			return clusteInfo{key, element.Server}, nil
		}
	}
	return clusteInfo{}, fmt.Errorf("unable to get cluster identifiers from kube_config")
}

// buildConfig is a helper function that builds configs from a kubeconfig filepath.
// If kubeconfigPath is not passed in we fallback to inClusterConfig.
// If inClusterConfig fails, we fallback to the default config.
// This is a copy of `clientcmd.BuildConfigFromFlags` of `client-go` but without the annoying
// klog messages that are not possible to be disabled.
func buildConfig(kubeconfigPath string) (*restclient.Config, error) {
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
	if kubeconfig != "" || getKubeConfigEnvironmentVariable() != "" {
		return false
	}
	return true
}

// DiscoverKubernetesNode figures out the Kubernetes node to use.
// If host is provided in the config use it directly.
// If beat is deployed in k8s cluster, use hostname of pod which is pod name to query pod meta for node name.
// If beat is deployed outside k8s cluster, use machine-id to match against k8s nodes for node name.
func DiscoverKubernetesNode(log *logp.Logger, host string, inCluster bool, client kubernetes.Interface) (node string) {
	if host != "" {
		log.Infof("kubernetes: Using node %s provided in the config", host)
		return host
	}
	ctx := context.TODO()
	if inCluster {
		ns, err := InClusterNamespace()
		if err != nil {
			log.Errorf("kubernetes: Couldn't get namespace when beat is in cluster with error: %+v", err.Error())
			return defaultNode
		}
		podName, err := os.Hostname()
		if err != nil {
			log.Errorf("kubernetes: Couldn't get hostname as beat pod name in cluster with error: %+v", err.Error())
			return defaultNode
		}
		log.Infof("kubernetes: Using pod name %s and namespace %s to discover kubernetes node", podName, ns)
		pod, err := client.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			log.Errorf("kubernetes: Querying for pod failed with error: %+v", err)
			return defaultNode
		}
		log.Infof("kubernetes: Using node %s discovered by in cluster pod node query", pod.Spec.NodeName)
		return pod.Spec.NodeName
	}

	mid := machineID()
	if mid == "" {
		log.Error("kubernetes: Couldn't collect info from any of the files in /etc/machine-id /var/lib/dbus/machine-id")
		return defaultNode
	}

	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Errorf("kubernetes: Querying for nodes failed with error: %+v", err)
		return defaultNode
	}
	for _, n := range nodes.Items {
		if n.Status.NodeInfo.MachineID == mid {
			name := n.GetObjectMeta().GetName()
			log.Infof("kubernetes: Using node %s discovered by machine-id matching", name)
			return name
		}
	}

	log.Warn("kubernetes: Couldn't discover node, using localhost as default")
	return defaultNode
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
	data, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
