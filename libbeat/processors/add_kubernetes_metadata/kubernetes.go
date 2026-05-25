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
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"

	k8sclient "k8s.io/client-go/kubernetes"

	"github.com/elastic/elastic-agent-autodiscover/kubernetes"
	"github.com/elastic/elastic-agent-autodiscover/kubernetes/metadata"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/shared"
)

const (
	timeout  = time.Second * 5
	selector = "kubernetes"
)

type kubernetesAnnotator struct {
	log                 *logp.Logger
	watcher             kubernetes.Watcher
	nsWatcher           kubernetes.Watcher
	nodeWatcher         kubernetes.Watcher
	rsWatcher           kubernetes.Watcher
	jobWatcher          kubernetes.Watcher
	indexers            *Indexers
	matchers            *Matchers
	cache               *cache
	kubernetesAvailable bool
	initOnce            sync.Once
	wg                  sync.WaitGroup
	cancelCtx           context.CancelFunc
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

func isKubernetesAvailable(client k8sclient.Interface, logger *logp.Logger) (bool, error) {
	server, err := client.Discovery().ServerVersion()
	if err != nil {
		return false, err
	}
	logger.Infof("add_kubernetes_metadata: kubernetes env detected, with version: %v", server)
	return true, nil
}

func isKubernetesAvailableWithTimeout(
	ctx context.Context,
	client k8sclient.Interface,
	waitMetadataTimeout time.Duration,
	waitMetadataRetryPeriod time.Duration,
	logger *logp.Logger,
) (bool, error) {
	ticker := time.NewTicker(waitMetadataRetryPeriod)
	defer ticker.Stop()

	var timeoutC <-chan time.Time
	if waitMetadataTimeout > 0 {
		timeout := time.NewTimer(waitMetadataTimeout)
		defer timeout.Stop()
		timeoutC = timeout.C
	}

	for {
		available, err := isKubernetesAvailable(client, logger)
		if available {
			return true, nil
		}

		select {
		case <-ctx.Done():
			return false, fmt.Errorf("context cancelled while waiting for kubernetes to be available: %w", ctx.Err())
		case <-timeoutC:
			logger.Errorf("add_kubernetes_metadata: could not detect kubernetes env: %v", err)
			return false, fmt.Errorf("timeout waiting for kubernetes to be available: %w", err)
		case <-ticker.C:
		}
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

	ctx, cancelCtx := context.WithCancel(context.Background())
	processor := &kubernetesAnnotator{
		log:       log,
		cache:     newCache(config.CleanupTimeout),
		cancelCtx: cancelCtx,
	}

	if config.WaitMetadata {
		err := processor.init(ctx, config, cfg)
		if !processor.kubernetesAvailable {
			cancelCtx()
			return nil, fmt.Errorf("add_kubernetes_metadata: %w", err)
		}
	} else {
		// complete processor's initialisation asynchronously to re-try on failing k8s client initialisations in case
		// the k8s node is not yet ready.
		processor.wg.Add(1)
		go func() {
			_ = processor.init(ctx, config, cfg)
			processor.wg.Done()
		}()
	}

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

func (k *kubernetesAnnotator) init(ctx context.Context, config kubeAnnotatorConfig, cfg *config.C) error {
	var k8sError error
	k.initOnce.Do(func() {
		var replicaSetWatcher, jobWatcher, namespaceWatcher, nodeWatcher kubernetes.Watcher

		// We initialise the use_kubeadm variable based on modules KubeAdm base configuration
		err := config.AddResourceMetadata.Namespace.SetBool("use_kubeadm", -1, config.KubeAdm)
		if err != nil {
			k.log.Errorf("couldn't set kubeadm variable for namespace due to error %+v", err)
		}
		err = config.AddResourceMetadata.Node.SetBool("use_kubeadm", -1, config.KubeAdm)
		if err != nil {
			k.log.Errorf("couldn't set kubeadm variable for node due to error %+v", err)
		}
		client, err := kubernetes.GetKubernetesClient(config.KubeConfig, config.KubeClientOptions)
		if err != nil {
			if kubernetes.IsInCluster(config.KubeConfig) {
				k.log.Debugf("Could not create kubernetes client using in_cluster config: %+v", err)
			} else if config.KubeConfig == "" {
				k.log.Debugf("Could not create kubernetes client using config: %v: %+v", os.Getenv("KUBECONFIG"), err)
			} else {
				k.log.Debugf("Could not create kubernetes client using config: %v: %+v", config.KubeConfig, err)
			}
			k8sError = err
			return
		}

		if available, err := isKubernetesAvailableWithTimeout(ctx, client, config.WaitMetadataTimeout, config.WaitMetadataRetryPeriod, k.log); !available {
			k8sError = err
			return
		}

		matchers := NewMatchers(config.Matchers, k.log)

		if matchers.Empty() {
			commonMsg := "Could not initialize kubernetes plugin with zero matcher plugins"
			k.log.Debug(commonMsg)
			k8sError = errors.New(commonMsg)
			return
		}

		k.matchers = matchers
		nd := &kubernetes.DiscoverKubernetesNodeParams{
			ConfigHost:  config.Node,
			Client:      client,
			IsInCluster: kubernetes.IsInCluster(config.KubeConfig),
			HostUtils:   &kubernetes.DefaultDiscoveryUtils{},
		}
		if config.Scope == "node" {
			config.Node, k8sError = kubernetes.DiscoverKubernetesNode(k.log, nd)
			if k8sError != nil {
				k.log.Errorf("Couldn't discover Kubernetes node: %v", k8sError)
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
			k8sError = err
			return
		}

		metaConf := config.AddResourceMetadata

		if metaConf.Node.Enabled() {
			nodeWatcher, k8sError = kubernetes.NewNamedWatcher("add_kubernetes_metadata_node", client, &kubernetes.Node{}, kubernetes.WatchOptions{
				SyncTimeout:  config.SyncPeriod,
				Node:         config.Node,
				HonorReSyncs: true,
			}, nil, k.log)
			if k8sError != nil {
				k.log.Errorf("couldn't create watcher for %T due to error %+v", &kubernetes.Node{}, k8sError)
			}
		}

		if metaConf.Namespace.Enabled() {
			namespaceWatcher, k8sError = kubernetes.NewNamedWatcher("add_kubernetes_metadata_namespace", client, &kubernetes.Namespace{}, kubernetes.WatchOptions{
				SyncTimeout:  config.SyncPeriod,
				Namespace:    config.Namespace,
				HonorReSyncs: true,
			}, nil, k.log)
			if k8sError != nil {
				k.log.Errorf("couldn't create watcher for %T due to error %+v", &kubernetes.Namespace{}, k8sError)
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
			replicaSetWatcher, k8sError = kubernetes.NewNamedMetadataWatcher(
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
			if k8sError != nil {
				k.log.Errorf("Error creating watcher for %T due to error %+v", &kubernetes.ReplicaSet{}, k8sError)
			}
			k.rsWatcher = replicaSetWatcher
		}
		if metaConf.CronJob {
			jobWatcher, k8sError = kubernetes.NewNamedWatcher("resource_metadata_enricher_job", client, &kubernetes.Job{}, kubernetes.WatchOptions{
				SyncTimeout:  config.SyncPeriod,
				Namespace:    config.Namespace,
				HonorReSyncs: true,
			}, nil, k.log)
			if k8sError != nil {
				k.log.Errorf("Error creating watcher for %T due to error %+v", &kubernetes.Job{}, k8sError)
			}
			k.jobWatcher = jobWatcher
		}

		// TODO: refactor the above section to a common function to be used by NeWPodEventer too
		metaGen := metadata.GetPodMetaGen(cfg, watcher, nodeWatcher, namespaceWatcher, replicaSetWatcher, jobWatcher, metaConf)

		k.indexers = NewIndexers(config.Indexers, metaGen)
		k.watcher = watcher
		k.kubernetesAvailable = true
		k.nodeWatcher = nodeWatcher
		k.nsWatcher = namespaceWatcher

		watcher.AddEventHandler(kubernetes.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod, _ := obj.(*kubernetes.Pod)
				k.addPod(pod)
			},
			UpdateFunc: func(obj interface{}) {
				pod, _ := obj.(*kubernetes.Pod)
				k.updatePod(pod)
			},
			DeleteFunc: func(obj interface{}) {
				pod, _ := obj.(*kubernetes.Pod)
				k.removePod(pod)
			},
		})

		// NOTE: order is important here since pod meta will include node meta and hence node.Store() should
		// be populated before trying to generate metadata for Pods.
		if k.nodeWatcher != nil {
			if err := k.nodeWatcher.Start(); err != nil {
				k.log.Debugf("Couldn't start node watcher: %v", err)
				return
			}
		}
		if k.nsWatcher != nil {
			if err := k.nsWatcher.Start(); err != nil {
				k.log.Debugf("Couldn't start namespace watcher: %v", err)
				return
			}
		}
		if k.rsWatcher != nil {
			if err := k.rsWatcher.Start(); err != nil {
				k.log.Debugf("Couldn't start replicaSet watcher: %v", err)
				return
			}
		}
		if k.jobWatcher != nil {
			if err := k.jobWatcher.Start(); err != nil {
				k.log.Debugf("Couldn't start job watcher: %v", err)
				return
			}
		}
		if err := watcher.Start(); err != nil {
			k.log.Debugf("Couldn't start pod watcher: %v", err)
			return
		}
	})

	return k8sError
}

// Run runs the processor that adds a field `kubernetes` to the event fields that
// contains a map with various Kubernetes metadata.
// This processor does not access or modify the `Meta` of the event.
func (k *kubernetesAnnotator) Run(event *beat.Event) (*beat.Event, error) {
	if kubernetesMetadataExist(event) {
		return event, nil
	}

	if !k.kubernetesAvailable {
		return event, nil
	}

	index := k.matchers.MetadataIndex(event.Fields)
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
	if k.cancelCtx != nil {
		k.cancelCtx()
	}
	k.wg.Wait()

	if k.watcher != nil {
		k.watcher.Stop()
	}
	if k.nodeWatcher != nil {
		k.nodeWatcher.Stop()
	}
	if k.nsWatcher != nil {
		k.nsWatcher.Stop()
	}
	if k.rsWatcher != nil {
		k.rsWatcher.Stop()
	}
	if k.jobWatcher != nil {
		k.jobWatcher.Stop()
	}
	if k.cache != nil {
		k.cache.stop()
	}

	return nil
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
