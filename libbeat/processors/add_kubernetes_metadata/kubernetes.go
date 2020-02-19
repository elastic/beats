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
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/kubernetes/metadata"

	k8sclient "k8s.io/client-go/kubernetes"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes/metadata"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/processors"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
)

const (
	timeout                = time.Second * 5
	selector               = "kubernetes"
	checkNodeReadyAttempts = 10
)

type kubernetesAnnotator struct {
	log                 *logp.Logger
	watcher             kubernetes.Watcher
	indexers            *Indexers
	matchers            *Matchers
	cache               *cache
	kubernetesAvailable bool
	initOnce            sync.Once
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

func isKubernetesAvailable(client k8sclient.Interface) (bool, error) {
	server, err := client.Discovery().ServerVersion()
	if err != nil {
		return false, err
	}
	logp.Info("%v: kubernetes env detected, with version: %v", "add_kubernetes_metadata", server)
	return true, nil
}

func isKubernetesAvailableWithRetry(client k8sclient.Interface) bool {
	connectionAttempts := 1
	for {
		kubernetesAvailable, err := isKubernetesAvailable(client)
		if kubernetesAvailable {
			return true
		}
		if connectionAttempts > checkNodeReadyAttempts {
			logp.Info("%v: could not detect kubernetes env: %v", "add_kubernetes_metadata", err)
			return false
		}
		time.Sleep(3 * time.Second)
		connectionAttempts += 1
	}
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

	// complete processor's initialisation asynchronously so as to re-try on failing k8s client initialisations in case
	// the k8s node is not yet ready.
	go processor.init(config, cfg)

	return processor, nil
}

func (k *kubernetesAnnotator) init(config kubeAnnotatorConfig, cfg *common.Config) {
	k.initOnce.Do(func() {
		client, err := kubernetes.GetKubernetesClient(config.KubeConfig)
		if err != nil {
			if kubernetes.IsInCluster(config.KubeConfig) {
				k.log.Debugf("Could not create kubernetes client using in_cluster config: %+v", err)
			} else if config.KubeConfig == "" {
				k.log.Debugf("Could not create kubernetes client using config: %v: %+v", os.Getenv("KUBECONFIG"), err)
			} else {
				k.log.Debugf("Could not create kubernetes client using config: %v: %+v", config.KubeConfig, err)
			}
			return
		}

		if !isKubernetesAvailableWithRetry(client) {
			return
		}

		matchers := NewMatchers(config.Matchers)

		if matchers.Empty() {
			k.log.Debugf("Could not initialize kubernetes plugin with zero matcher plugins")
			return
		}

		k.matchers = matchers

		config.Host = kubernetes.DiscoverKubernetesNode(k.log, config.Host, kubernetes.IsInCluster(config.KubeConfig), client)

		k.log.Debugf("Initializing a new Kubernetes watcher using host: %s", config.Host)

		watcher, err := kubernetes.NewWatcher(client, &kubernetes.Pod{}, kubernetes.WatchOptions{
			SyncTimeout: config.SyncPeriod,
			Node:        config.Host,
			Namespace:   config.Namespace,
		}, nil)
		if err != nil {
			k.log.Errorf("Couldn't create kubernetes watcher for %T", &kubernetes.Pod{})
			return
		}

		metaGen := metadata.NewPodMetadataGenerator(cfg, watcher.Store(), nil, nil)
		k.indexers = NewIndexers(config.Indexers, metaGen)
		k.watcher = watcher
		k.kubernetesAvailable = true

		watcher.AddEventHandler(kubernetes.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod := obj.(*kubernetes.Pod)
				k.log.Debugf("Adding kubernetes pod: %s/%s", pod.GetNamespace(), pod.GetName())
				k.addPod(pod)
			},
			UpdateFunc: func(obj interface{}) {
				pod := obj.(*kubernetes.Pod)
				k.log.Debugf("Updating kubernetes pod: %s/%s", pod.GetNamespace(), pod.GetName())
				k.updatePod(pod)
			},
			DeleteFunc: func(obj interface{}) {
				pod := obj.(*kubernetes.Pod)
				k.log.Debugf("Removing pod: %s/%s", pod.GetNamespace(), pod.GetName())
				k.removePod(pod)
			},
		})

		if err := watcher.Start(); err != nil {
			k.log.Debugf("add_kubernetes_metadata", "Couldn't start watcher: %v", err)
			return
		}
	})
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
