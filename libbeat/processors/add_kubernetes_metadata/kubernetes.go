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

//go:build linux || darwin || windows

package add_kubernetes_metadata

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"

	k8sclient "k8s.io/client-go/kubernetes"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/shared"
	"github.com/elastic/beats/v7/pkg/autodiscover/kubernetes"
	"github.com/elastic/beats/v7/pkg/autodiscover/kubernetes/metadata"
)

const (
	timeout                = time.Second * 5
	selector               = "kubernetes"
	checkNodeReadyAttempts = 10
)

// initializedState holds the fields that init populates and that Run or Close read after startup.
type initializedState struct {
	watcher     kubernetes.Watcher
	nsWatcher   kubernetes.Watcher
	nodeWatcher kubernetes.Watcher
	rsWatcher   kubernetes.Watcher
	jobWatcher  kubernetes.Watcher
	matchers    *Matchers
}

type kubernetesAnnotator struct {
	log      *logp.Logger
	state    atomic.Pointer[initializedState]
	cache    *cache
	initOnce sync.Once
}

func init() {
	processors.RegisterPlugin("add_kubernetes_metadata", shared.New(New))

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

// kubernetesMetadataExist checks whether an event is already enriched with kubernetes metadata
func kubernetesMetadataExist(event *beat.Event) bool {
	if _, err := event.GetValue("kubernetes"); err != nil {
		return false
	}
	return true
}

// New constructs a new add_kubernetes_metadata processor.
func New(cfg *config.C, log *logp.Logger) (beat.Processor, error) {
	config, err := newProcessorConfig(cfg, Indexing)
	if err != nil {
		return nil, err
	}

	log = log.Named(selector).With("libbeat.processor", "add_kubernetes_metadata")
	processor := &kubernetesAnnotator{
		log:   log,
		cache: newCache(config.CleanupTimeout),
	}

	// complete processor's initialisation asynchronously to re-try on failing k8s client initialisations in case
	// the k8s node is not yet ready.
	go processor.init(config, cfg)

	return processor, nil
}

func newProcessorConfig(cfg *config.C, register *Register) (kubeAnnotatorConfig, error) {
	var config kubeAnnotatorConfig
	err := cfg.Unpack(&config)
	if err != nil {
		return config, fmt.Errorf("fail to unpack the kubernetes configuration: %w", err)
	}

	// Load and append default indexer configs
	if config.DefaultIndexers.Enabled {
		config.Indexers = append(config.Indexers, register.GetDefaultIndexerConfigs()...)
	}

	// Load and append default matcher configs
	if config.DefaultMatchers.Enabled {
		config.Matchers = append(config.Matchers, register.GetDefaultMatcherConfigs()...)
	}

	return config, nil
}

func (k *kubernetesAnnotator) init(config kubeAnnotatorConfig, cfg *config.C) {
	k.initOnce.Do(func() {
		var replicaSetWatcher, jobWatcher, namespaceWatcher, nodeWatcher kubernetes.Watcher

		client, err := kubernetes.GetKubernetesClient(config.KubeConfig, config.KubeClientOptions)
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

		matchers := NewMatchers(config.Matchers, k.log)

		if matchers.Empty() {
			k.log.Debugf("Could not initialize kubernetes plugin with zero matcher plugins")
			return
		}

		nd := &kubernetes.DiscoverKubernetesNodeParams{
			ConfigHost:  config.Node,
			Client:      client,
			IsInCluster: kubernetes.IsInCluster(config.KubeConfig),
			HostUtils:   &kubernetes.DefaultDiscoveryUtils{},
		}
		if config.Scope == "node" {
			config.Node, err = kubernetes.DiscoverKubernetesNode(k.log, nd)
			if err != nil {
				k.log.Errorf("Couldn't discover Kubernetes node: %v", err)
				return
			}
			k.log.Debugf("Initializing a new Kubernetes watcher using host: %s", config.Node)
		}

		watcher, err := kubernetes.NewNamedWatcher("add_kubernetes_metadata_pod", client, &kubernetes.Pod{}, kubernetes.WatchOptions{
			SyncTimeout:  config.SyncPeriod,
			Node:         config.Node,
			Namespace:    config.Namespace,
			HonorReSyncs: true,
		}, nil, k.log)
		if err != nil {
			k.log.Errorf("Couldn't create kubernetes watcher for %T", &kubernetes.Pod{})
			return
		}

		// Copy add_resource_metadata and its sub-configs before propagating use_kubeadm, so init
		// never mutates the caller-owned config.
		metaConf := *config.AddResourceMetadata
		metaConf.Node = configWithKubeadm(k.log, metaConf.Node, config.KubeAdm)
		metaConf.Namespace = configWithKubeadm(k.log, metaConf.Namespace, config.KubeAdm)

		if metaConf.Node.Enabled() {
			nodeWatcher, err = kubernetes.NewNamedWatcher("add_kubernetes_metadata_node", client, &kubernetes.Node{}, kubernetes.WatchOptions{
				SyncTimeout:  config.SyncPeriod,
				Node:         config.Node,
				HonorReSyncs: true,
			}, nil, k.log)
			if err != nil {
				k.log.Errorf("couldn't create watcher for %T due to error %+v", &kubernetes.Node{}, err)
			}
		}

		if metaConf.Namespace.Enabled() {
			namespaceWatcher, err = kubernetes.NewNamedWatcher("add_kubernetes_metadata_namespace", client, &kubernetes.Namespace{}, kubernetes.WatchOptions{
				SyncTimeout:  config.SyncPeriod,
				Namespace:    config.Namespace,
				HonorReSyncs: true,
			}, nil, k.log)
			if err != nil {
				k.log.Errorf("couldn't create watcher for %T due to error %+v", &kubernetes.Namespace{}, err)
			}
		}

		// Resource is Pod, so we need to create watchers for Replicasets and Jobs that it might belong to
		// in order to be able to retrieve 2nd layer Owner metadata like in case of:
		// Deployment -> Replicaset -> Pod
		// CronJob -> job -> Pod
		if metaConf.Deployment {
			metadataClient, err := kubernetes.GetKubernetesMetadataClient(config.KubeConfig, config.KubeClientOptions)
			if err != nil {
				k.log.Errorf("Error creating metadata client due to error %+v", err)
			}
			replicaSetWatcher, err = kubernetes.NewNamedMetadataWatcher(
				"resource_metadata_enricher_rs",
				client,
				metadataClient,
				schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "replicasets"},
				kubernetes.WatchOptions{
					SyncTimeout:  config.SyncPeriod,
					Namespace:    config.Namespace,
					HonorReSyncs: true,
				},
				nil,
				metadata.RemoveUnnecessaryReplicaSetData,
				k.log,
			)
			if err != nil {
				k.log.Errorf("Error creating watcher for %T due to error %+v", &kubernetes.ReplicaSet{}, err)
			}
		}
		if metaConf.CronJob {
			jobWatcher, err = kubernetes.NewNamedWatcher("resource_metadata_enricher_job", client, &kubernetes.Job{}, kubernetes.WatchOptions{
				SyncTimeout:  config.SyncPeriod,
				Namespace:    config.Namespace,
				HonorReSyncs: true,
			}, nil, k.log)
			if err != nil {
				k.log.Errorf("Error creating watcher for %T due to error %+v", &kubernetes.Job{}, err)
			}
		}

		// TODO: refactor the above section to a common function to be used by NeWPodEventer too
		metaGen := metadata.GetPodMetaGen(cfg, watcher, nodeWatcher, namespaceWatcher, replicaSetWatcher, jobWatcher, &metaConf)

		indexers := NewIndexers(config.Indexers, metaGen)

		watcher.AddEventHandler(kubernetes.ResourceEventHandlerFuncs{
			AddFunc: func(obj any) {
				pod, _ := obj.(*kubernetes.Pod)
				k.addPod(indexers, pod)
			},
			UpdateFunc: func(obj any) {
				pod, _ := obj.(*kubernetes.Pod)
				k.updatePod(indexers, pod)
			},
			DeleteFunc: func(obj any) {
				pod, _ := obj.(*kubernetes.Pod)
				k.removePod(indexers, pod)
			},
		})

		// Publish the fully constructed state atomically before starting the watchers.
		k.state.Store(&initializedState{
			watcher:     watcher,
			nsWatcher:   namespaceWatcher,
			nodeWatcher: nodeWatcher,
			rsWatcher:   replicaSetWatcher,
			jobWatcher:  jobWatcher,
			matchers:    matchers,
		})

		// NOTE: order is important here since pod meta will include node meta and hence node.Store() should
		// be populated before trying to generate metadata for Pods.
		if nodeWatcher != nil {
			if err := nodeWatcher.Start(); err != nil {
				k.log.Debugf("Couldn't start node watcher: %v", err)
				return
			}
		}
		if namespaceWatcher != nil {
			if err := namespaceWatcher.Start(); err != nil {
				k.log.Debugf("Couldn't start namespace watcher: %v", err)
				return
			}
		}
		if replicaSetWatcher != nil {
			if err := replicaSetWatcher.Start(); err != nil {
				k.log.Debugf("Couldn't start replicaSet watcher: %v", err)
				return
			}
		}
		if jobWatcher != nil {
			if err := jobWatcher.Start(); err != nil {
				k.log.Debugf("Couldn't start job watcher: %v", err)
				return
			}
		}
		if err := watcher.Start(); err != nil {
			k.log.Debugf("Couldn't start pod watcher: %v", err)
			return
		}
	})
}

// Run runs the processor that adds a field `kubernetes` to the event fields that
// contains a map with various Kubernetes metadata.
// This processor does not access or modify the `Meta` of the event.
func (k *kubernetesAnnotator) Run(event *beat.Event) (*beat.Event, error) {
	// A nil state means init has not published yet (still running or kubernetes unavailable); the
	// load pairs with the Store in init for a race-free read.
	state := k.state.Load()
	if state == nil {
		return event, nil
	}
	if kubernetesMetadataExist(event) {
		return event, nil
	}

	index := state.matchers.MetadataIndex(event.Fields)
	if index == "" {
		k.log.Debug("No container match string, not adding kubernetes data")
		return event, nil
	}

	metadata := k.cache.get(index)
	if metadata == nil {
		return event, nil
	}

	// One full clone for the kubernetes field; one cheap sub-map clone for the OCI
	// container field. This replaces the original three full clones.
	kubeMeta := metadata.Clone()

	// Build the OCI container field by cloning only the container sub-map —
	// much cheaper than cloning the full metadata. Transform it in place:
	// drop container.name and rewrite container.image -> container.image.name.
	if containerVal, err := kubeMeta.GetValue("kubernetes.container"); err == nil {
		if cm, ok := containerVal.(mapstr.M); ok {
			ociContainer := cm.Clone()
			_ = ociContainer.Delete("name")
			if img, imgErr := ociContainer.GetValue("image"); imgErr == nil {
				_ = ociContainer.Delete("image")
				ociContainer["image"] = mapstr.M{"name": img}
			}
			event.Fields.DeepUpdate(mapstr.M{"container": ociContainer})
		}
	}

	// Remove container fields that belong only in the OCI section before writing
	// kubernetes metadata to the event. container.name is intentionally kept here
	// to match original behaviour.
	_ = kubeMeta.Delete("kubernetes.container.id")
	_ = kubeMeta.Delete("kubernetes.container.runtime")
	_ = kubeMeta.Delete("kubernetes.container.image")
	event.Fields.DeepUpdate(kubeMeta)

	return event, nil
}

func (k *kubernetesAnnotator) Close() error {
	// ensure there are no goroutines leaking
	// after the processor has been closed
	k.initOnce.Do(func() {})
	if state := k.state.Load(); state != nil {
		if state.watcher != nil {
			state.watcher.Stop()
		}
		if state.nodeWatcher != nil {
			state.nodeWatcher.Stop()
		}
		if state.nsWatcher != nil {
			state.nsWatcher.Stop()
		}
		if state.rsWatcher != nil {
			state.rsWatcher.Stop()
		}
		if state.jobWatcher != nil {
			state.jobWatcher.Stop()
		}
	}
	if k.cache != nil {
		k.cache.stop()
	}
	return nil
}

func (k *kubernetesAnnotator) addPod(indexers *Indexers, pod *kubernetes.Pod) {
	metadata := indexers.GetMetadata(pod)
	for _, m := range metadata {
		k.cache.set(m.Index, m.Data)
	}
}

func (k *kubernetesAnnotator) updatePod(indexers *Indexers, pod *kubernetes.Pod) {
	k.removePod(indexers, pod)

	// Add it again only if it is not being deleted
	if pod.GetObjectMeta().GetDeletionTimestamp() != nil {
		return
	}

	k.addPod(indexers, pod)
}

func (k *kubernetesAnnotator) removePod(indexers *Indexers, pod *kubernetes.Pod) {
	indexes := indexers.GetIndexes(pod)
	for _, idx := range indexes {
		k.cache.delete(idx)
	}
}

func (*kubernetesAnnotator) String() string {
	return "add_kubernetes_metadata"
}

// configWithKubeadm sets use_kubeadm to the given value without mutating cfg.
func configWithKubeadm(log *logp.Logger, cfg *config.C, kubeAdm bool) *config.C {
	if cfg == nil {
		return nil
	}
	clone, err := config.MergeConfigs(cfg)
	if err != nil {
		// Copying a valid config does not fail in practice; keep the original untouched.
		log.Errorf("couldn't copy add_resource_metadata config to set use_kubeadm: %+v", err)
		return cfg
	}
	if err := clone.SetBool("use_kubeadm", -1, kubeAdm); err != nil {
		log.Errorf("couldn't set use_kubeadm variable due to error %+v", err)
	}
	return clone
}
