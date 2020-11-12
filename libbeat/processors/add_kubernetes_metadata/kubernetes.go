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

	k8sclient "k8s.io/client-go/kubernetes"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes/metadata"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/processors"
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
	config, err := newProcessorConfig(cfg, Indexing)
	if err != nil {
		return nil, err
	}

	log := logp.NewLogger(selector).With("libbeat.processor", "add_kubernetes_metadata")
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

func newProcessorConfig(cfg *common.Config, register *Register) (kubeAnnotatorConfig, error) {
	config := defaultKubernetesAnnotatorConfig()

	err := cfg.Unpack(&config)
	if err != nil {
		return config, fmt.Errorf("fail to unpack the kubernetes configuration: %s", err)
	}

	//Load and append default indexer configs
	if config.DefaultIndexers.Enabled {
		config.Indexers = append(config.Indexers, register.GetDefaultIndexerConfigs()...)
	}

	//Load and append default matcher configs
	if config.DefaultMatchers.Enabled {
		config.Matchers = append(config.Matchers, register.GetDefaultMatcherConfigs()...)
	}

	return config, nil
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

		metaConf := config.AddResourceMetadata
		if metaConf == nil {
			metaConf = metadata.GetDefaultResourceMetadataConfig()
		}

		options := kubernetes.WatchOptions{
			SyncTimeout: config.SyncPeriod,
			Node:        "",
		}
		if config.Namespace != "" {
			options.Namespace = config.Namespace
		}
		nodeWatcher, err := kubernetes.NewWatcher(client, &kubernetes.Node{}, options, nil)
		if err != nil {
			k.log.Errorf("couldn't create watcher for %T due to error %+v", &kubernetes.Node{}, err)
		}
		namespaceWatcher, err := kubernetes.NewWatcher(client, &kubernetes.Namespace{}, kubernetes.WatchOptions{
			SyncTimeout: config.SyncPeriod,
		}, nil)
		if err != nil {
			k.log.Errorf("couldn't create watcher for %T due to error %+v", &kubernetes.Namespace{}, err)
		}
		// TODO: refactor the above section to a common function to be used by NeWPodEventer too
		metaGen := metadata.GetPodMetaGen(cfg, watcher, nodeWatcher, namespaceWatcher, metaConf)

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

		// NOTE: order is important here since pod meta will include node meta and hence node.Store() should
		// be populated before trying to generate metadata for Pods.
		if err := nodeWatcher.Start(); err != nil {
			k.log.Debugf("add_kubernetes_metadata", "Couldn't start node watcher: %v", err)
			return
		}
		if err := namespaceWatcher.Start(); err != nil {
			k.log.Debugf("add_kubernetes_metadata", "Couldn't start namespace watcher: %v", err)
			return
		}
		if err := watcher.Start(); err != nil {
			k.log.Debugf("add_kubernetes_metadata", "Couldn't start pod watcher: %v", err)
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
		k.log.Debug("No container match string, not adding kubernetes data")
		return event, nil
	}

	k.log.Debugf("Using the following index key %s", index)
	metadata := k.cache.get(index)
	if metadata == nil {
		k.log.Debugf("Index key %s did not match any of the cached resources", index)
		return event, nil
	}

	metaClone := metadata.Clone()
	metaClone.Delete("container.name")
	containerImage, err := metadata.GetValue("container.image")
	if err == nil {
		metaClone.Delete("container.image")
		metaClone.Put("container.image.name", containerImage)
	}
	cmeta, err := metaClone.Clone().GetValue("container")
	if err == nil {
		event.Fields.DeepUpdate(common.MapStr{
			"container": cmeta,
		})
	}

	kubeMeta := metadata.Clone()
	kubeMeta.Delete("container.id")
	kubeMeta.Delete("container.runtime")
	event.Fields.DeepUpdate(common.MapStr{
		"kubernetes": kubeMeta,
	})

	return event, nil
}

func (k *kubernetesAnnotator) Close() error {
	if k.watcher != nil {
		k.watcher.Stop()
	}
	if k.cache != nil {
		k.cache.stop()
	}
	return nil
}

func (k *kubernetesAnnotator) addPod(pod *kubernetes.Pod) {
	metadata := k.indexers.GetMetadata(pod)
	for _, m := range metadata {
		k.log.Debugf("Created index %s for pod %s/%s", m.Index, pod.GetNamespace(), pod.GetName())
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
