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
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"encoding/json"
	"strings"
	"path"
	"net"
	"context"
	"time"

	"gopkg.in/yaml.v2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"
	//restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	//clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/common/safemapstr"


    s "github.com/elastic/beats/v7/libbeat/common/schema"
       c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
)

// MetaGen allows creation of metadata from either Kubernetes resources or their Resource names.
type MetaGen interface {
	// Generate generates metadata for a given resource
	Generate(kubernetes.Resource, ...FieldOptions) common.MapStr
	// GenerateFromName generates metadata for a given resource based on it's name
	GenerateFromName(string, ...FieldOptions) common.MapStr
	// GenerateK8s generates metadata for a given resource
	GenerateK8s(kubernetes.Resource, ...FieldOptions) common.MapStr
	// GenerateK8s generates metadata for a given resource
	GenerateECS(kubernetes.Resource) common.MapStr
}

// FieldOptions allows additional enrichment to be done on top of existing metadata
type FieldOptions func(common.MapStr)

type ClusterInfo struct {
	Url  string
	Name string
}

type KubeConfig struct {
	Clusters   []Clusters `yaml:"clusters"`
}

type Clusters struct {
	Cluster Cluster `yaml:"cluster"`
}

type Cluster struct {
	Server string `yaml:"server"`
}

// WithFields FieldOption allows adding specific fields into the generated metadata
func WithFields(key string, value interface{}) FieldOptions {
	return func(meta common.MapStr) {
		safemapstr.Put(meta, key, value)
	}
}

// WithLabels FieldOption allows adding labels under sub-resource(kind)
// example if kind=namespace namespace.labels key will be added
func WithLabels(kind string) FieldOptions {
	return func(meta common.MapStr) {
		safemapstr.Put(meta, strings.ToLower(kind)+".labels", meta["labels"])
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


func GetKubernetesClusterIdentifier(cfg *common.Config, client k8sclient.Interface) (ClusterInfo, error) {
	// try with kubeadm-config
	var config Config
	config.Unmarshal(cfg)
	//clusterInfo, err := getClusterInfoFromKubeConfigFile(config.KubeConfig)
	//if err == nil {
	//	return clusterInfo, nil
	//}
	// try with kubeadm-config
	//clusterInfo, err = getClusterInfoFromKubeadmConfigMap(client)
	//if err == nil {
	//	return clusterInfo, nil
	//}
	// try with GKE metadata
	clusterInfo, err := getClusterInfoFromGKEMetadata(cfg)
	if err == nil {
		return clusterInfo, nil
	}
	return ClusterInfo{}, fmt.Errorf("unable to retrieve cluster identifiers")
}

func getClusterInfoFromGKEMetadata(cfg *common.Config) (ClusterInfo, error) {
	kubeConfigURI := "http://metadata.google.internal/computeMetadata/v1/instance/attributes/kubeconfig?alt-json"
	clusterNameURI := "http://metadata.google.internal/computeMetadata/v1/instance/attributes/cluster-name?alt=json"
	gceHeaders := map[string]string{"Metadata-Flavor": "Google"}
	//gceSchema := func(m map[string]interface{}) common.MapStr {
	//	fmt.Println("inside schema func:")
	//	fmt.Println(m)
	//	out := common.MapStr{
	//		"service": common.MapStr{
	//			"name": "GCE",
	//		},
	//	}
	//
	//	trimLeadingPath := func(key string) {
	//		v, err := out.GetValue(key)
	//		if err != nil {
	//			return
	//		}
	//		p, ok := v.(string)
	//		if !ok {
	//			return
	//		}
	//		out.Put(key, path.Base(p))
	//	}
	//
	//	if instance, ok := m["instance"].(map[string]interface{}); ok {
	//		s.Schema{
	//			"instance": s.Object{
	//				"id":   c.StrFromNum("id"),
	//				"name": c.Str("name"),
	//			},
	//			"machine": s.Object{
	//				"type": c.Str("machineType"),
	//			},
	//			"availability_zone": c.Str("zone"),
	//		}.ApplyTo(out, instance)
	//		trimLeadingPath("machine.type")
	//		trimLeadingPath("availability_zone")
	//	}
	//
	//	if project, ok := m["project"].(map[string]interface{}); ok {
	//		s.Schema{
	//			"project": s.Object{
	//				"id": c.Str("projectId"),
	//			},
	//			"account": s.Object{
	//				"id": c.Str("projectId"),
	//			},
	//		}.ApplyTo(out, project)
	//	}
	//
	//	return out
	//}

	client := http.Client{
		Timeout: 1 * time.Minute,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			DialContext: (&net.Dialer{
				Timeout:   1 * time.Minute,
				KeepAlive: 0,
			}).DialContext,
		},
	}

	fmt.Println("Going to fetch metadataaaaaa")
	ctx, cancel := context.WithTimeout(context.TODO(), 1 * time.Minute)
	defer cancel()
	clusterName := fetchRaw(ctx, client, clusterNameURI, gceHeaders)
	fmt.Println("here is the clusterName")
	fmt.Println(clusterName)
	kubeConfig := fetchRaw(ctx, client, kubeConfigURI, gceHeaders)
	fmt.Println("here is the kubeConfig")
	fmt.Println(kubeConfig)
	cc := &KubeConfig{}
	err := yaml.Unmarshal([]byte(kubeConfig), cc)
	if err != nil {
		return ClusterInfo{}, err
	}
	fmt.Println("here is the clusterServer")
	fmt.Println(cc.Clusters[0].Cluster.Server)

	return ClusterInfo{}, fmt.Errorf("unable to get cluster identifiers from GKE metadata")
}

type ClusterConfiguration struct {
	ControlPlaneEndpoint string `yaml:"controlPlaneEndpoint"`
	ClusterName          string `yaml:"clusterName"`
}

func getClusterInfoFromKubeadmConfigMap(client k8sclient.Interface) (ClusterInfo, error) {
	clusterInfo := ClusterInfo{}
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
			return ClusterInfo{key, element.Server}, nil
		}
	}
	return ClusterInfo{}, fmt.Errorf("unable to get cluster identifiers from kube_config")
}


func fetchRaw(
	ctx context.Context,
	client http.Client,
	url string,
	headers map[string]string,
) string {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	req = req.WithContext(ctx)

	rsp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != http.StatusOK {
		fmt.Println(rsp.StatusCode)
		return ""
	}

	all, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return ""
	}

	var metadata string
	dec := json.NewDecoder(bytes.NewReader(all))
	dec.UseNumber()
	err = dec.Decode(&metadata)
	if err != nil {
		return ""
	}
	return metadata
}
