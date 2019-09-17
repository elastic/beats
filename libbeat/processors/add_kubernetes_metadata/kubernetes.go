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

// +build linux darwin windows

package add_kubernetes_metadata

import (
	"fmt"
	"time"

	k8sclient "k8s.io/client-go/kubernetes"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/kubernetes"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	jsprocessor "github.com/elastic/beats/libbeat/processors/script/javascript/module/processor"
)

const (
	timeout = time.Second * 5
)

type kubernetesAnnotator struct {
	watcher             kubernetes.Watcher
	indexers            *Indexers
	matchers            *Matchers
	cache               *cache
	kubernetesAvailable bool
}

func init() {
	processors.RegisterPlugin("add_kubernetes_metadata", New)
	jsprocessor.RegisterPlugin("AddKubernetesMetadata", New)

	// Register default indexers
	Indexing.AddIndexer(PodNameIndexerName, NewPodNameIndexer)
	Indexing.AddIndexer(PodUIDIndexerName, NewPodUIDIndexer)
	Indexing.AddIndexer(ContainerIndexerName, NewContainerIndexer)
	Indexing.AddIndexer(IPPortIndexerName, NewIPPortIndexer)
	Indexing.AddMatcher(FieldMatcherName, NewFieldMatcher)
	Indexing.AddMatcher(FieldFormatMatcherName, NewFieldFormatMatcher)
}

func isKubernetesAvailable(client k8sclient.Interface) bool {
	server, err := client.Discovery().ServerVersion()
	if err != nil {
		logp.Info("%v: could not detect kubernetes env: %v", "add_kubernetes_metadata", err)
		return false
	}
	logp.Info("%v: kubernetes env detected, with version: %v", "add_kubernetes_metadata", server)
	return true
}

// New constructs a new add_kubernetes_metadata processor.
func New(cfg *common.Config) (processors.Processor, error) {
	config := defaultKubernetesAnnotatorConfig()

	err := cfg.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the kubernetes configuration: %s", err)
	}

	//Load default indexer configs
	if config.DefaultIndexers.Enabled == true {
		Indexing.RLock()
		for key, cfg := range Indexing.GetDefaultIndexerConfigs() {
			config.Indexers = append(config.Indexers, map[string]common.Config{key: cfg})
		}
		Indexing.RUnlock()
	}

	//Load default matcher configs
	if config.DefaultMatchers.Enabled == true {
		Indexing.RLock()
		for key, cfg := range Indexing.GetDefaultMatcherConfigs() {
			config.Matchers = append(config.Matchers, map[string]common.Config{key: cfg})
		}
		Indexing.RUnlock()
	}

	metaGen, err := kubernetes.NewMetaGenerator(cfg)
	if err != nil {
		return nil, err
	}

	processor := &kubernetesAnnotator{
		cache:               newCache(config.CleanupTimeout),
		kubernetesAvailable: false,
	}

	client, err := kubernetes.GetKubernetesClient(config.KubeConfig)
	if err != nil {
		if kubernetes.IsInCluster(config.KubeConfig) {
			logp.Debug("kubernetes", "%v: could not create kubernetes client using in_cluster config", "add_kubernetes_metadata")
		} else {
			logp.Debug("kubernetes", "%v: could not create kubernetes client using config: %v", "add_kubernetes_metadata", config.KubeConfig)
		}
		return processor, nil
	}

	if !isKubernetesAvailable(client) {
		return processor, nil
	}

	processor.indexers = NewIndexers(config.Indexers, metaGen)

	matchers := NewMatchers(config.Matchers)

	if matchers.Empty() {
		logp.Debug("kubernetes", "%v: could not initialize kubernetes plugin with zero matcher plugins", "add_kubernetes_metadata")
		return processor, nil
	}

	processor.matchers = matchers

	config.Host = kubernetes.DiscoverKubernetesNode(config.Host, kubernetes.IsInCluster(config.KubeConfig), client)

	logp.Debug("kubernetes", "Using host: %s", config.Host)
	logp.Debug("kubernetes", "Initializing watcher")

	watcher, err := kubernetes.NewWatcher(client, &kubernetes.Pod{}, kubernetes.WatchOptions{
		SyncTimeout: config.SyncPeriod,
		Node:        config.Host,
		Namespace:   config.Namespace,
	})
	if err != nil {
		logp.Err("kubernetes: Couldn't create watcher for %T", &kubernetes.Pod{})
		return nil, err
	}

	processor.watcher = watcher
	processor.kubernetesAvailable = true

	watcher.AddEventHandler(kubernetes.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			processor.addPod(obj.(*kubernetes.Pod))
		},
		UpdateFunc: func(obj interface{}) {
			processor.removePod(obj.(*kubernetes.Pod))
			processor.addPod(obj.(*kubernetes.Pod))
		},
		DeleteFunc: func(obj interface{}) {
			processor.removePod(obj.(*kubernetes.Pod))
		},
	})

	if err := watcher.Start(); err != nil {
		return nil, err
	}

	return processor, nil
}

func (k *kubernetesAnnotator) Run(event *beat.Event) (*beat.Event, error) {
	if !k.kubernetesAvailable {
		return event, nil
	}
	index := k.matchers.MetadataIndex(event.Fields)
	if index == "" {
		return event, nil
	}

	metadata := k.cache.get(index)
	if metadata == nil {
		return event, nil
	}

	event.Fields.DeepUpdate(common.MapStr{
		"kubernetes": metadata.Clone(),
	})

	return event, nil
}

func (k *kubernetesAnnotator) addPod(pod *kubernetes.Pod) {
	metadata := k.indexers.GetMetadata(pod)
	for _, m := range metadata {
		k.cache.set(m.Index, m.Data)
	}
}

func (k *kubernetesAnnotator) removePod(pod *kubernetes.Pod) {
	indexes := k.indexers.GetIndexes(pod)
	for _, idx := range indexes {
		k.cache.delete(idx)
	}
}

func (*kubernetesAnnotator) String() string {
	return "add_kubernetes_metadata"
}
