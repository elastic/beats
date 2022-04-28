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

package util

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/kubernetes/metadata"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	conf "github.com/elastic/elastic-agent-libs/config"
)

// Enricher takes Kubernetes events and enrich them with k8s metadata
type Enricher interface {
	// Start will start the Kubernetes watcher on the first call, does nothing on the rest
	// errors are logged as warning
	Start()

	// Stop will stop the Kubernetes watcher
	Stop()

	// Enrich the given list of events
	Enrich([]mapstr.M)
}

type kubernetesConfig struct {
	KubeConfig        string                       `config:"kube_config"`
	KubeClientOptions kubernetes.KubeClientOptions `config:"kube_client_options"`

	Node       string        `config:"node"`
	SyncPeriod time.Duration `config:"sync_period"`

	// AddMetadata enables enriching metricset events with metadata from the API server
	AddMetadata         bool                                `config:"add_metadata"`
	AddResourceMetadata *metadata.AddResourceMetadataConfig `config:"add_resource_metadata"`
	Namespace           string                              `config:"namespace"`
}

type enricher struct {
	sync.RWMutex
	metadata            map[string]mapstr.M
	index               func(mapstr.M) string
	watcher             kubernetes.Watcher
	watchersStarted     bool
	watchersStartedLock sync.Mutex
	namespaceWatcher    kubernetes.Watcher
	nodeWatcher         kubernetes.Watcher
	isPod               bool
}

const selector = "kubernetes"

// NewResourceMetadataEnricher returns an Enricher configured for kubernetes resource events
func NewResourceMetadataEnricher(
	base mb.BaseMetricSet,
	res kubernetes.Resource,
	nodeScope bool) Enricher {

	config := validatedConfig(base)
	if config == nil {
		logp.Info("Kubernetes metricset enriching is disabled")
		return &nilEnricher{}
	}

	watcher, nodeWatcher, namespaceWatcher := getResourceMetadataWatchers(config, res, nodeScope)

	if watcher == nil {
		return &nilEnricher{}
	}

	// GetPodMetaGen requires cfg of type Config
	commonMetaConfig := metadata.Config{}
	if err := base.Module().UnpackConfig(&commonMetaConfig); err != nil {
		logp.Err("Error initializing Kubernetes metadata enricher: %s", err)
		return &nilEnricher{}
	}
	cfg, _ := conf.NewConfigFrom(&commonMetaConfig)

	metaGen := metadata.NewResourceMetadataGenerator(cfg, watcher.Client())
	podMetaGen := metadata.GetPodMetaGen(cfg, watcher, nodeWatcher, namespaceWatcher, config.AddResourceMetadata)

	namespaceMeta := metadata.NewNamespaceMetadataGenerator(config.AddResourceMetadata.Namespace, namespaceWatcher.Store(), watcher.Client())
	serviceMetaGen := metadata.NewServiceMetadataGenerator(cfg, watcher.Store(), namespaceMeta, watcher.Client())
	enricher := buildMetadataEnricher(watcher, nodeWatcher, namespaceWatcher,
		// update
		func(m map[string]mapstr.M, r kubernetes.Resource) {
			accessor, _ := meta.Accessor(r)
			id := join(accessor.GetNamespace(), accessor.GetName())

			switch r := r.(type) {
			case *kubernetes.Pod:
				m[id] = podMetaGen.Generate(r)

			case *kubernetes.Node:
				// Report node allocatable resources to PerfMetrics cache
				name := r.GetObjectMeta().GetName()
				if cpu, ok := r.Status.Capacity["cpu"]; ok {
					if q, err := resource.ParseQuantity(cpu.String()); err == nil {
						PerfMetrics.NodeCoresAllocatable.Set(name, float64(q.MilliValue())/1000)
					}
				}
				if memory, ok := r.Status.Capacity["memory"]; ok {
					if q, err := resource.ParseQuantity(memory.String()); err == nil {
						PerfMetrics.NodeMemAllocatable.Set(name, float64(q.Value()))
					}
				}

				m[id] = metaGen.Generate("node", r)

			case *kubernetes.Deployment:
				m[id] = metaGen.Generate("deployment", r)
			case *kubernetes.Job:
				m[id] = metaGen.Generate("job", r)
			case *kubernetes.CronJob:
				m[id] = metaGen.Generate("cronjob", r)
			case *kubernetes.Service:
				m[id] = serviceMetaGen.Generate(r)
			case *kubernetes.StatefulSet:
				m[id] = metaGen.Generate("statefulset", r)
			case *kubernetes.Namespace:
				m[id] = metaGen.Generate("namespace", r)
			case *kubernetes.ReplicaSet:
				m[id] = metaGen.Generate("replicaset", r)
			default:
				m[id] = metaGen.Generate(r.GetObjectKind().GroupVersionKind().Kind, r)
			}
		},
		// delete
		func(m map[string]mapstr.M, r kubernetes.Resource) {
			accessor, _ := meta.Accessor(r)
			id := join(accessor.GetNamespace(), accessor.GetName())
			delete(m, id)
		},
		// index
		func(e mapstr.M) string {
			return join(getString(e, mb.ModuleDataKey+".namespace"), getString(e, "name"))
		},
	)

	// Configure the enricher for Pods, so pod specific metadata ends up in the right place when
	// calling Enrich
	if _, ok := res.(*kubernetes.Pod); ok {
		enricher.isPod = true
	}

	return enricher
}

// NewContainerMetadataEnricher returns an Enricher configured for container events
func NewContainerMetadataEnricher(
	base mb.BaseMetricSet,
	nodeScope bool) Enricher {

	config := validatedConfig(base)
	if config == nil {
		logp.Info("Kubernetes metricset enriching is disabled")
		return &nilEnricher{}
	}

	watcher, nodeWatcher, namespaceWatcher := getResourceMetadataWatchers(config, &kubernetes.Pod{}, nodeScope)
	if watcher == nil {
		return &nilEnricher{}
	}

	commonMetaConfig := metadata.Config{}
	if err := base.Module().UnpackConfig(&commonMetaConfig); err != nil {
		logp.Err("Error initializing Kubernetes metadata enricher: %s", err)
		return &nilEnricher{}
	}
	cfg, _ := conf.NewConfigFrom(&commonMetaConfig)

	metaGen := metadata.GetPodMetaGen(cfg, watcher, nodeWatcher, namespaceWatcher, config.AddResourceMetadata)

	enricher := buildMetadataEnricher(watcher, nodeWatcher, namespaceWatcher,
		// update
		func(m map[string]mapstr.M, r kubernetes.Resource) {
			pod, ok := r.(*kubernetes.Pod)
			if !ok {
				base.Logger().Debugf("Error while casting event: %s", ok)
			}
			meta := metaGen.Generate(pod)

			statuses := make(map[string]*kubernetes.PodContainerStatus)
			mapStatuses := func(s []kubernetes.PodContainerStatus) {
				for i := range s {
					statuses[s[i].Name] = &s[i]
				}
			}
			mapStatuses(pod.Status.ContainerStatuses)
			mapStatuses(pod.Status.InitContainerStatuses)
			for _, container := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
				cuid := ContainerUID(pod.GetObjectMeta().GetNamespace(), pod.GetObjectMeta().GetName(), container.Name)

				// Report container limits to PerfMetrics cache
				if cpu, ok := container.Resources.Limits["cpu"]; ok {
					if q, err := resource.ParseQuantity(cpu.String()); err == nil {
						PerfMetrics.ContainerCoresLimit.Set(cuid, float64(q.MilliValue())/1000)
					}
				}
				if memory, ok := container.Resources.Limits["memory"]; ok {
					if q, err := resource.ParseQuantity(memory.String()); err == nil {
						PerfMetrics.ContainerMemLimit.Set(cuid, float64(q.Value()))
					}
				}

				if s, ok := statuses[container.Name]; ok {
					// Extracting id and runtime ECS fields from ContainerID
					// which is in the form of <container.runtime>://<container.id>
					split := strings.Index(s.ContainerID, "://")
					if split != -1 {
						ShouldPut(meta, "container.id", s.ContainerID[split+3:], base.Logger())

						ShouldPut(meta, "container.runtime", s.ContainerID[:split], base.Logger())
					}
				}
				id := join(pod.GetObjectMeta().GetNamespace(), pod.GetObjectMeta().GetName(), container.Name)
				m[id] = meta
			}
		},
		// delete
		func(m map[string]mapstr.M, r kubernetes.Resource) {
			pod, ok := r.(*kubernetes.Pod)
			if !ok {
				base.Logger().Debugf("Error while casting event: %s", ok)
			}
			for _, container := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
				id := join(pod.ObjectMeta.GetNamespace(), pod.GetObjectMeta().GetName(), container.Name)
				delete(m, id)
			}
		},
		// index
		func(e mapstr.M) string {
			return join(getString(e, mb.ModuleDataKey+".namespace"), getString(e, mb.ModuleDataKey+".pod.name"), getString(e, "name"))
		},
	)

	return enricher
}

func getResourceMetadataWatchers(config *kubernetesConfig, resource kubernetes.Resource, nodeScope bool) (kubernetes.Watcher, kubernetes.Watcher, kubernetes.Watcher) {
	client, err := kubernetes.GetKubernetesClient(config.KubeConfig, config.KubeClientOptions)
	if err != nil {
		logp.Err("Error creating Kubernetes client: %s", err)
		return nil, nil, nil
	}

	options := kubernetes.WatchOptions{
		SyncTimeout: config.SyncPeriod,
		Namespace:   config.Namespace,
	}

	log := logp.NewLogger(selector)

	// Watch objects in the node only
	if nodeScope {
		nd := &kubernetes.DiscoverKubernetesNodeParams{
			ConfigHost:  config.Node,
			Client:      client,
			IsInCluster: kubernetes.IsInCluster(config.KubeConfig),
			HostUtils:   &kubernetes.DefaultDiscoveryUtils{},
		}
		options.Node, err = kubernetes.DiscoverKubernetesNode(log, nd)
		if err != nil {
			logp.Err("Couldn't discover kubernetes node: %s", err)
			return nil, nil, nil
		}
	}

	log.Debugf("Initializing a new Kubernetes watcher using host: %v", config.Node)

	watcher, err := kubernetes.NewNamedWatcher("resource_metadata_enricher", client, resource, options, nil)
	if err != nil {
		logp.Err("Error initializing Kubernetes watcher: %s", err)
		return nil, nil, nil
	}

	nodeWatcher, err := kubernetes.NewNamedWatcher("resource_metadata_enricher_node", client, &kubernetes.Node{}, options, nil)
	if err != nil {
		logp.Err("Error creating watcher for %T due to error %+v", &kubernetes.Node{}, err)
		return watcher, nil, nil
	}

	namespaceWatcher, err := kubernetes.NewNamedWatcher("resource_metadata_enricher_namespace", client, &kubernetes.Namespace{}, kubernetes.WatchOptions{
		SyncTimeout: config.SyncPeriod,
	}, nil)
	if err != nil {
		logp.Err("Error creating watcher for %T due to error %+v", &kubernetes.Namespace{}, err)
		return watcher, nodeWatcher, nil
	}

	return watcher, nodeWatcher, namespaceWatcher
}

func GetDefaultDisabledMetaConfig() *kubernetesConfig {
	return &kubernetesConfig{
		AddMetadata: false,
	}
}

func validatedConfig(base mb.BaseMetricSet) *kubernetesConfig {
	config := kubernetesConfig{
		AddMetadata:         true,
		SyncPeriod:          time.Minute * 10,
		AddResourceMetadata: metadata.GetDefaultResourceMetadataConfig(),
	}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil
	}

	// Return nil if metadata enriching is disabled:
	if !config.AddMetadata {
		return nil
	}
	return &config
}

func getString(m mapstr.M, key string) string {
	val, err := m.GetValue(key)
	if err != nil {
		return ""
	}

	str, _ := val.(string)
	return str
}

func join(fields ...string) string {
	return strings.Join(fields, ":")
}

func buildMetadataEnricher(
	watcher kubernetes.Watcher,
	nodeWatcher kubernetes.Watcher,
	namespaceWatcher kubernetes.Watcher,
	update func(map[string]mapstr.M, kubernetes.Resource),
	delete func(map[string]mapstr.M, kubernetes.Resource),
	index func(e mapstr.M) string) *enricher {

	enricher := enricher{
		metadata:         map[string]mapstr.M{},
		index:            index,
		watcher:          watcher,
		nodeWatcher:      nodeWatcher,
		namespaceWatcher: namespaceWatcher,
	}

	watcher.AddEventHandler(kubernetes.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			enricher.Lock()
			defer enricher.Unlock()
			update(enricher.metadata, obj.(kubernetes.Resource))
		},
		UpdateFunc: func(obj interface{}) {
			enricher.Lock()
			defer enricher.Unlock()
			update(enricher.metadata, obj.(kubernetes.Resource))
		},
		DeleteFunc: func(obj interface{}) {
			enricher.Lock()
			defer enricher.Unlock()
			delete(enricher.metadata, obj.(kubernetes.Resource))
		},
	})

	return &enricher
}

func (m *enricher) Start() {
	m.watchersStartedLock.Lock()
	defer m.watchersStartedLock.Unlock()
	if !m.watchersStarted {
		if m.nodeWatcher != nil {
			if err := m.nodeWatcher.Start(); err != nil {
				logp.Warn("Error starting node watcher: %s", err)
			}
		}

		if m.namespaceWatcher != nil {
			if err := m.namespaceWatcher.Start(); err != nil {
				logp.Warn("Error starting namespace watcher: %s", err)
			}
		}

		err := m.watcher.Start()
		if err != nil {
			logp.Warn("Error starting Kubernetes watcher: %s", err)
		}
		m.watchersStarted = true
	}
}

func (m *enricher) Stop() {
	m.watchersStartedLock.Lock()
	defer m.watchersStartedLock.Unlock()
	if m.watchersStarted {
		m.watcher.Stop()

		if m.namespaceWatcher != nil {
			m.namespaceWatcher.Stop()
		}

		if m.nodeWatcher != nil {
			m.nodeWatcher.Stop()
		}

		m.watchersStarted = false
	}
}

func (m *enricher) Enrich(events []mapstr.M) {
	m.RLock()
	defer m.RUnlock()
	for _, event := range events {
		if meta := m.metadata[m.index(event)]; meta != nil {
			k8s, err := meta.GetValue("kubernetes")
			if err != nil {
				continue
			}
			k8sMeta, ok := k8s.(mapstr.M)
			if !ok {
				continue
			}

			if m.isPod {
				// apply pod meta at metricset level
				if podMeta, ok := k8sMeta["pod"].(mapstr.M); ok {
					event.DeepUpdate(podMeta)
				}

				// don't apply pod metadata to module level
				k8sMeta = k8sMeta.Clone()
				delete(k8sMeta, "pod")
			}
			ecsMeta := meta.Clone()
			err = ecsMeta.Delete("kubernetes")
			if err != nil {
				logp.Debug("kubernetes", "Failed to delete field '%s': %s", "kubernetes", err)
			}

			event.DeepUpdate(mapstr.M{
				mb.ModuleDataKey: k8sMeta,
				"meta":           ecsMeta,
			})
		}
	}
}

type nilEnricher struct{}

func (*nilEnricher) Start()            {}
func (*nilEnricher) Stop()             {}
func (*nilEnricher) Enrich([]mapstr.M) {}

func CreateEvent(event mapstr.M, namespace string) (mb.Event, error) {
	var moduleFieldsMapStr mapstr.M
	moduleFields, ok := event[mb.ModuleDataKey]
	var err error
	if ok {
		moduleFieldsMapStr, ok = moduleFields.(mapstr.M)
		if !ok {
			err = fmt.Errorf("error trying to convert '%s' from event to mapstr.M", mb.ModuleDataKey)
		}
	}
	delete(event, mb.ModuleDataKey)

	e := mb.Event{
		MetricSetFields: event,
		ModuleFields:    moduleFieldsMapStr,
		Namespace:       namespace,
	}

	// add root-level fields like ECS fields
	var metaFieldsMapStr mapstr.M
	metaFields, ok := event["meta"]
	if ok {
		metaFieldsMapStr, ok = metaFields.(mapstr.M)
		if !ok {
			err = fmt.Errorf("error trying to convert '%s' from event to mapstr.M", "meta")
		}
		delete(event, "meta")
		if len(metaFieldsMapStr) > 0 {
			e.RootFields = metaFieldsMapStr
		}
	}
	return e, err
}

func ShouldPut(event mapstr.M, field string, value interface{}, logger *logp.Logger) {
	_, err := event.Put(field, value)
	if err != nil {
		logger.Debugf("Failed to put field '%s' with value '%s': %s", field, value, err)
	}
}

func ShouldDelete(event mapstr.M, field string, logger *logp.Logger) {
	err := event.Delete(field)
	if err != nil {
		logger.Debugf("Failed to delete field '%s': %s", field, err)
	}
}
