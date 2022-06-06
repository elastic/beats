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

package metadata

import (
	"context"
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/common/safemapstr"
)

// MetaGen allows creation of metadata from either Kubernetes resources or their Resource names.
type MetaGen interface {
	// Generate generates metadata for a given resource.
	// Metadata map is formed in the following format:
	// {
	//    "kubernetes": GenerateK8s(),
	//    "some.ecs.field": "asdf, // populated by GenerateECS()
	// }
	// This method is called in top level and returns the complete map of metadata.
	Generate(kubernetes.Resource, ...FieldOptions) common.MapStr
	// GenerateFromName generates metadata for a given resource based on it's name
	GenerateFromName(string, ...FieldOptions) common.MapStr
	// GenerateK8s generates kubernetes metadata for a given resource
	GenerateK8s(kubernetes.Resource, ...FieldOptions) common.MapStr
	// GenerateECS generates ECS metadata for a given resource
	GenerateECS(kubernetes.Resource) common.MapStr
}

// FieldOptions allows additional enrichment to be done on top of existing metadata
type FieldOptions func(common.MapStr)

type ClusterInfo struct {
	Url  string
	Name string
}

type ClusterConfiguration struct {
	ControlPlaneEndpoint string `yaml:"controlPlaneEndpoint"`
	ClusterName          string `yaml:"clusterName"`
}

// WithFields FieldOption allows adding specific fields into the generated metadata
func WithFields(key string, value interface{}) FieldOptions {
	return func(meta common.MapStr) {
		safemapstr.Put(meta, key, value)
	}
}

// WithMetadata FieldOption allows adding labels and annotations under sub-resource(kind)
// example if kind=namespace namespace.labels key will be added
func WithMetadata(kind string) FieldOptions {
	return func(meta common.MapStr) {
		if meta["labels"] != nil {
			safemapstr.Put(meta, strings.ToLower(kind)+".labels", meta["labels"])
		}
		if meta["annotations"] != nil {
			safemapstr.Put(meta, strings.ToLower(kind)+".annotations", meta["annotations"])
		}
	}
}

// GetPodMetaGen is a wrapper function that creates a metaGen for pod resource and has embeeded
// nodeMetaGen and namespaceMetaGen
func GetPodMetaGen(
	cfg *common.Config,
	podWatcher kubernetes.Watcher,
	nodeWatcher kubernetes.Watcher,
	namespaceWatcher kubernetes.Watcher,
	metaConf *AddResourceMetadataConfig) MetaGen {

	var nodeMetaGen, namespaceMetaGen MetaGen
	if nodeWatcher != nil && metaConf.Node.Enabled() {
		nodeMetaGen = NewNodeMetadataGenerator(metaConf.Node, nodeWatcher.Store(), nodeWatcher.Client())
	}
	if namespaceWatcher != nil && metaConf.Namespace.Enabled() {
		namespaceMetaGen = NewNamespaceMetadataGenerator(metaConf.Namespace, namespaceWatcher.Store(), namespaceWatcher.Client())
	}
	metaGen := NewPodMetadataGenerator(cfg, podWatcher.Store(), podWatcher.Client(), nodeMetaGen, namespaceMetaGen)

	return metaGen
}

// GetKubernetesClusterIdentifier returns ClusterInfo for k8s if available
func GetKubernetesClusterIdentifier(cfg *common.Config, client k8sclient.Interface) (ClusterInfo, error) {
	// try with kube config file
	var config Config
	config.Unmarshal(cfg)
	clusterInfo, err := getClusterInfoFromKubeConfigFile(config.KubeConfig)
	if err == nil {
		return clusterInfo, nil
	}
	// try with kubeadm-config configmap
	clusterInfo, err = getClusterInfoFromKubeadmConfigMap(client)
	if err == nil {
		return clusterInfo, nil
	}
	return ClusterInfo{}, fmt.Errorf("unable to retrieve cluster identifiers")
}

func getClusterInfoFromKubeadmConfigMap(client k8sclient.Interface) (ClusterInfo, error) {
	clusterInfo := ClusterInfo{}
	if client == nil {
		return clusterInfo, fmt.Errorf("unable to get cluster identifiers from kubeadm-config")
	}
	cm, err := client.CoreV1().ConfigMaps("kube-system").Get(context.TODO(), "kubeadm-config", metav1.GetOptions{})
	if err != nil {
		return clusterInfo, fmt.Errorf("unable to get cluster identifiers from kubeadm-config: %+v", err)
	}
	p, ok := cm.Data["ClusterConfiguration"]
	if !ok {
		return clusterInfo, fmt.Errorf("unable to get cluster identifiers from ClusterConfiguration")
	}

	cc := &ClusterConfiguration{}
	err = yaml.Unmarshal([]byte(p), cc)
	if err != nil {
		return ClusterInfo{}, err
	}
	if cc.ClusterName != "" {
		clusterInfo.Name = cc.ClusterName
	}
	if cc.ControlPlaneEndpoint != "" {
		clusterInfo.Url = cc.ControlPlaneEndpoint
	}

	return clusterInfo, nil
}

func getClusterInfoFromKubeConfigFile(kubeconfig string) (ClusterInfo, error) {
	if kubeconfig == "" {
		kubeconfig = kubernetes.GetKubeConfigEnvironmentVariable()
	}

	if kubeconfig == "" {
		return ClusterInfo{}, fmt.Errorf("unable to get cluster identifiers from kube_config from env")
	}

	cfg, err := kubernetes.BuildConfig(kubeconfig)
	if err != nil {
		return ClusterInfo{}, fmt.Errorf("unable to build kube config due to error: %+v", err)
	}

	kube_cfg, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return ClusterInfo{}, fmt.Errorf("unable to load kube_config due to error: %+v", err)
	}

	for key, element := range kube_cfg.Clusters {
		if element.Server == cfg.Host {
			return ClusterInfo{element.Server, key}, nil
		}
	}
	return ClusterInfo{}, fmt.Errorf("unable to get cluster identifiers from kube_config")
}
