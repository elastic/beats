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
	"os"
	"time"

	"github.com/elastic/beats/libbeat/common/kubernetes/metadata"

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
	log                 *logp.Logger
	watcher             kubernetes.Watcher
	indexers            *Indexers
	matchers            *Matchers
	cache               *cache
	kubernetesAvailable bool
}

const (
	selector          = "kubernetes"
	nodeReadyAttempts = 10
)

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
	log := logp.NewLogger(selector).With("libbeat.processor", "add_kubernetes_metadata")

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

	processor := &kubernetesAnnotator{
		log:                 log,
		cache:               newCache(config.CleanupTimeout),
		kubernetesAvailable: false,
	}

	// complete processor's initialisation asyncronously so as to re-try on failing k8s client initialisations in case
	// the k8s node is not yet ready.
	go func(processor *kubernetesAnnotator) {
		client, err := kubernetes.GetKubernetesClient(config.KubeConfig)
		if err != nil {
			if kubernetes.IsInCluster(config.KubeConfig) {
				log.Debugf("Could not create kubernetes client using in_cluster config: %+v", err)
			} else if config.KubeConfig == "" {
				log.Debugf("Could not create kubernetes client using config: %v: %+v", os.Getenv("KUBECONFIG"), err)
			} else {
				log.Debugf("Could not create kubernetes client using config: %v: %+v", config.KubeConfig, err)
			}
			return
		}

		connectionAttempts := 1
		for ok := true; ok; ok = !isKubernetesAvailable(client) {
			if connectionAttempts > nodeReadyAttempts {
				return
			}
			time.Sleep(3 * time.Second)
			connectionAttempts += 1
		}

		matchers := NewMatchers(config.Matchers)

		if matchers.Empty() {
			log.Debugf("Could not initialize kubernetes plugin with zero matcher plugins")
			return
		}

		processor.matchers = matchers

		config.Host = kubernetes.DiscoverKubernetesNode(log, config.Host, kubernetes.IsInCluster(config.KubeConfig), client)

		log.Debug("Initializing a new Kubernetes watcher using host: %s", config.Host)

		watcher, err := kubernetes.NewWatcher(client, &kubernetes.Pod{}, kubernetes.WatchOptions{
			SyncTimeout: config.SyncPeriod,
			Node:        config.Host,
			Namespace:   config.Namespace,
		}, nil)
		if err != nil {
			log.Errorf("Couldn't create kubernetes watcher for %T", &kubernetes.Pod{})
			return
		}

		metaGen := metadata.NewPodMetadataGenerator(cfg, watcher.Store(), nil, nil)
		processor.indexers = NewIndexers(config.Indexers, metaGen)
		processor.watcher = watcher
		processor.kubernetesAvailable = true

		watcher.AddEventHandler(kubernetes.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod := obj.(*kubernetes.Pod)
				log.Debugf("Adding kubernetes pod: %s/%s", pod.GetNamespace(), pod.GetName())
				processor.addPod(pod)
			},
			UpdateFunc: func(obj interface{}) {
				pod := obj.(*kubernetes.Pod)
				log.Debugf("Updating kubernetes pod: %s/%s", pod.GetNamespace(), pod.GetName())
				processor.updatePod(pod)
			},
			DeleteFunc: func(obj interface{}) {
				pod := obj.(*kubernetes.Pod)
				log.Debugf("Removing pod: %s/%s", pod.GetNamespace(), pod.GetName())
				processor.removePod(pod)
			},
		})

		if err := watcher.Start(); err != nil {
			return
		}
	}(processor)

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

func (k *kubernetesAnnotator) updatePod(pod *kubernetes.Pod) {
	k.removePod(pod)

	// Add it again only if it is not being deleted
	if pod.GetObjectMeta().GetDeletionTimestamp() != nil {
		k.log.Debugf("Removing kubernetes pod being terminated: %s/%s", pod.GetNamespace(), pod.GetName())
		return
	}

	k.addPod(pod)
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
