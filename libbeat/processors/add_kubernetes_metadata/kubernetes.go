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
	"math"
	"math/rand"
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
	timeout    = time.Second * 5
	maxBackoff = 30 * time.Second
	maxRetries = 10
)

type kubernetesAnnotator struct {
	watcher             kubernetes.Watcher
	indexers            *Indexers
	matchers            *Matchers
	cache               *cache
	kubernetesAvailable bool
	cfg                 *common.Config
	config              *kubeAnnotatorConfig
	nextRetry           time.Time
	retryCount          byte
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

	processor := &kubernetesAnnotator{
		cache:               newCache(config.CleanupTimeout),
		cfg:                 cfg,
		config:              &config,
		kubernetesAvailable: false,
	}

	return processor, processor.initKubernetes()
}

func (k *kubernetesAnnotator) initKubernetes() error {
	client, err := kubernetes.GetKubernetesClient(k.config.KubeConfig)
	if err != nil {
		if kubernetes.IsInCluster(k.config.KubeConfig) {
			logp.Debug("kubernetes", "%v: could not create kubernetes client using in_cluster config: %v", "add_kubernetes_metadata", err)
		} else if k.config.KubeConfig == "" {
			logp.Debug("kubernetes", "%v: could not create kubernetes client using config: %v: %v", "add_kubernetes_metadata", os.Getenv("KUBECONFIG"), err)
		} else {
			logp.Debug("kubernetes", "%v: could not create kubernetes client using config: %v: %v", "add_kubernetes_metadata", k.config.KubeConfig, err)
		}
		return nil
	}

	if !isKubernetesAvailable(client) {
		k.updateNextRetry()
		return nil
	}

	matchers := NewMatchers(k.config.Matchers)

	if matchers.Empty() {
		logp.Debug("kubernetes", "%v: could not initialize kubernetes plugin with zero matcher plugins", "add_kubernetes_metadata")
		return nil
	}

	k.matchers = matchers

	k.config.Host = kubernetes.DiscoverKubernetesNode(k.config.Host, kubernetes.IsInCluster(k.config.KubeConfig), client)

	logp.Debug("kubernetes", "Initializing a new Kubernetes watcher using host: %s", k.config.Host)

	watcher, err := kubernetes.NewWatcher(client, &kubernetes.Pod{}, kubernetes.WatchOptions{
		SyncTimeout: k.config.SyncPeriod,
		Node:        k.config.Host,
		Namespace:   k.config.Namespace,
	}, nil)
	if err != nil {
		logp.Err("kubernetes: Couldn't create watcher for %T", &kubernetes.Pod{})
		return err
	}

	metaGen := metadata.NewPodMetadataGenerator(k.cfg, watcher.Store(), nil, nil)
	k.indexers = NewIndexers(k.config.Indexers, metaGen)
	k.watcher = watcher
	k.kubernetesAvailable = true

	watcher.AddEventHandler(kubernetes.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := obj.(*kubernetes.Pod)
			logp.Debug("kubernetes", "%v: adding pod: %s/%s", "add_kubernetes_metadata", pod.GetNamespace(), pod.GetName())
			k.addPod(pod)
		},
		UpdateFunc: func(obj interface{}) {
			pod := obj.(*kubernetes.Pod)
			logp.Debug("kubernetes", "%v: updating pod: %s/%s", "add_kubernetes_metadata", pod.GetNamespace(), pod.GetName())
			k.updatePod(pod)
		},
		DeleteFunc: func(obj interface{}) {
			pod := obj.(*kubernetes.Pod)
			logp.Debug("kubernetes", "%v: removing pod: %s/%s", "add_kubernetes_metadata", pod.GetNamespace(), pod.GetName())
			k.removePod(pod)
		},
	})

	if err := watcher.Start(); err != nil {
		return err
	}

	return nil
}

func (k *kubernetesAnnotator) Run(event *beat.Event) (*beat.Event, error) {
	if !k.kubernetesAvailable {
		// Check if we should try again
		if k.retryCount <= maxRetries && k.nextRetry.Before(time.Now()) {
			k.initKubernetes()
		}

		// If it's still not available, pass
		if !k.kubernetesAvailable {
			return event, nil
		}
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

func (k *kubernetesAnnotator) updateNextRetry() {
	nextBackoff := time.Duration(math.Pow(2, float64(k.retryCount))+rand.Float64()) * time.Second
	if nextBackoff < maxBackoff {
		k.nextRetry = time.Now().Add(nextBackoff)
	} else {
		k.nextRetry = time.Now().Add(maxBackoff)
	}

	k.retryCount++
}

func (k *kubernetesAnnotator) updatePod(pod *kubernetes.Pod) {
	k.removePod(pod)

	// Add it again only if it is not being deleted
	if pod.GetObjectMeta().GetDeletionTimestamp() != nil {
		logp.Debug("kubernetes", "%v: removing pod being terminated: %s/%s", "add_kubernetes_metadata", pod.GetNamespace(), pod.GetName())
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
