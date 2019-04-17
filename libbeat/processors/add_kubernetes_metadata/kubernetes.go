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

package add_kubernetes_metadata

import (
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/kubernetes"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
)

const (
	timeout = time.Second * 5
)

type kubernetesAnnotator struct {
	watcher  kubernetes.Watcher
	indexers *Indexers
	matchers *Matchers
	cache    *cache
}

func init() {
	processors.RegisterPlugin("add_kubernetes_metadata", New)

	// Register default indexers
	Indexing.AddIndexer(PodNameIndexerName, NewPodNameIndexer)
	Indexing.AddIndexer(PodUIDIndexerName, NewPodUIDIndexer)
	Indexing.AddIndexer(ContainerIndexerName, NewContainerIndexer)
	Indexing.AddIndexer(IPPortIndexerName, NewIPPortIndexer)
	Indexing.AddMatcher(FieldMatcherName, NewFieldMatcher)
	Indexing.AddMatcher(FieldFormatMatcherName, NewFieldFormatMatcher)
}

// New constructs a new add_kubernetes_metadata processor.
func New(cfg *common.Config) (processors.Processor, error) {
	config := defaultKubernetesAnnotatorConfig()

	err := cfg.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the kubernetes configuration: %s", err)
	}

	err = validate(config)
	if err != nil {
		return nil, err
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

	indexers := NewIndexers(config.Indexers, metaGen)

	matchers := NewMatchers(config.Matchers)

	if matchers.Empty() {
		return nil, fmt.Errorf("Can not initialize kubernetes plugin with zero matcher plugins")
	}

	client, err := kubernetes.GetKubernetesClient(config.InCluster, config.KubeConfig)
	if err != nil {
		return nil, err
	}

	config.Host = kubernetes.DiscoverKubernetesNode(config.Host, config.InCluster, client)

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

	processor := &kubernetesAnnotator{
		watcher:  watcher,
		indexers: indexers,
		matchers: matchers,
		cache:    newCache(config.CleanupTimeout),
	}

	watcher.AddEventHandler(kubernetes.ResourceEventHandlerFuncs{
		AddFunc: func(obj kubernetes.Resource) {
			processor.addPod(obj.(*kubernetes.Pod))
		},
		UpdateFunc: func(obj kubernetes.Resource) {
			processor.removePod(obj.(*kubernetes.Pod))
			processor.addPod(obj.(*kubernetes.Pod))
		},
		DeleteFunc: func(obj kubernetes.Resource) {
			processor.removePod(obj.(*kubernetes.Pod))
		},
	})

	if err := watcher.Start(); err != nil {
		return nil, err
	}

	return processor, nil
}

func (k *kubernetesAnnotator) Run(event *beat.Event) (*beat.Event, error) {
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

func validate(config kubeAnnotatorConfig) error {
	if !config.InCluster && config.KubeConfig == "" {
		return errors.New("`kube_config` path can't be empty when in_cluster is set to false")
	}
	return nil
}
