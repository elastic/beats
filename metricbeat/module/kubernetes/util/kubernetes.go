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
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	k8sclient "k8s.io/client-go/kubernetes"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"

	kubernetes2 "github.com/elastic/beats/v7/libbeat/autodiscover/providers/kubernetes"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-autodiscover/kubernetes"
	"github.com/elastic/elastic-agent-autodiscover/kubernetes/metadata"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
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
	replicasetWatcher   kubernetes.Watcher
	jobWatcher          kubernetes.Watcher
	isPod               bool
}

const selector = "kubernetes"

const (
	PodResource                   = "pod"
	ServiceResource               = "service"
	DeploymentResource            = "deployment"
	ReplicaSetResource            = "replicaset"
	StatefulSetResource           = "statefulset"
	DaemonSetResource             = "daemonset"
	JobResource                   = "job"
	NodeResource                  = "node"
	CronJobResource               = "cronjob"
	PersistentVolumeResource      = "persistentvolume"
	PersistentVolumeClaimResource = "persistentvolumeclaim"
	StorageClassResource          = "storageclass"
	NamespaceResource             = "state_namespace"
)

func getResource(resourceName string) kubernetes.Resource {
	switch resourceName {
	case PodResource:
		return &kubernetes.Pod{}
	case ServiceResource:
		return &kubernetes.Service{}
	case DeploymentResource:
		return &kubernetes.Deployment{}
	case ReplicaSetResource:
		return &kubernetes.ReplicaSet{}
	case StatefulSetResource:
		return &kubernetes.StatefulSet{}
	case DaemonSetResource:
		return &kubernetes.DaemonSet{}
	case JobResource:
		return &kubernetes.Job{}
	case CronJobResource:
		return &kubernetes.CronJob{}
	case PersistentVolumeResource:
		return &kubernetes.PersistentVolume{}
	case PersistentVolumeClaimResource:
		return &kubernetes.PersistentVolumeClaim{}
	case StorageClassResource:
		return &kubernetes.StorageClass{}
	case NodeResource:
		return &kubernetes.Node{}
	case NamespaceResource:
		return &kubernetes.Namespace{}
	default:
		return nil
	}
}

// NewResourceMetadataEnricher returns an Enricher configured for kubernetes resource events
func NewResourceMetadataEnricher(
	base mb.BaseMetricSet,
	resourceName string,
	metricsRepo *MetricsRepo,
	nodeScope bool) Enricher {

	var replicaSetWatcher, jobWatcher kubernetes.Watcher

	config, err := GetValidatedConfig(base)
	if err != nil {
		logp.Info("Kubernetes metricset enriching is disabled")
		return &nilEnricher{}
	}

	res := getResource(resourceName)
	if res == nil {
		return &nilEnricher{}
	}

	client, err := kubernetes.GetKubernetesClient(config.KubeConfig, config.KubeClientOptions)
	if err != nil {
		logp.Err("Error creating Kubernetes client: %s", err)
		return &nilEnricher{}
	}

	watcher, nodeWatcher, namespaceWatcher := getResourceMetadataWatchers(config, res, client, nodeScope)

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

	// if Resource is Pod then we need to create watchers for Replicasets and Jobs that it might belongs to
	// in order to be able to retrieve 2nd layer Owner metadata like in case of:
	// Deployment -> Replicaset -> Pod
	// CronJob -> job -> Pod
	if resourceName == PodResource {
		if config.AddResourceMetadata.Deployment {
			replicaSetWatcher, err = kubernetes.NewNamedWatcher("resource_metadata_enricher_rs", client, &kubernetes.ReplicaSet{}, kubernetes.WatchOptions{
				SyncTimeout: config.SyncPeriod,
			}, nil)
			if err != nil {
				logp.Err("Error creating watcher for %T due to error %+v", &kubernetes.ReplicaSet{}, err)
				return &nilEnricher{}
			}
		}

		if config.AddResourceMetadata.CronJob {
			jobWatcher, err = kubernetes.NewNamedWatcher("resource_metadata_enricher_job", client, &kubernetes.Job{}, kubernetes.WatchOptions{
				SyncTimeout: config.SyncPeriod,
			}, nil)
			if err != nil {
				logp.Err("Error creating watcher for %T due to error %+v", &kubernetes.Job{}, err)
				return &nilEnricher{}
			}
		}
	}

	podMetaGen := metadata.GetPodMetaGen(cfg, watcher, nodeWatcher, namespaceWatcher, replicaSetWatcher, jobWatcher, config.AddResourceMetadata)

	namespaceMeta := metadata.NewNamespaceMetadataGenerator(config.AddResourceMetadata.Namespace, namespaceWatcher.Store(), watcher.Client())
	serviceMetaGen := metadata.NewServiceMetadataGenerator(cfg, watcher.Store(), namespaceMeta, watcher.Client())

	metaGen := metadata.NewNamespaceAwareResourceMetadataGenerator(cfg, watcher.Client(), namespaceMeta)

	enricher := buildMetadataEnricher(watcher, nodeWatcher, namespaceWatcher, replicaSetWatcher, jobWatcher,
		// update
		func(m map[string]mapstr.M, r kubernetes.Resource) {
			accessor, _ := meta.Accessor(r)
			id := join(accessor.GetNamespace(), accessor.GetName()) //nolint:all

			switch r := r.(type) {
			case *kubernetes.Pod:
				m[id] = podMetaGen.Generate(r)

			case *kubernetes.Node:
				nodeName := r.GetObjectMeta().GetName()
				metrics := NewNodeMetrics()
				if cpu, ok := r.Status.Capacity["cpu"]; ok {
					if q, err := resource.ParseQuantity(cpu.String()); err == nil {
						metrics.CoresAllocatable = NewFloat64Metric(float64(q.MilliValue()) / 1000)
					}
				}
				if memory, ok := r.Status.Capacity["memory"]; ok {
					if q, err := resource.ParseQuantity(memory.String()); err == nil {
						metrics.MemoryAllocatable = NewFloat64Metric(float64(q.Value()))
					}
				}
				nodeStore, _ := metricsRepo.AddNodeStore(nodeName)
				nodeStore.SetNodeMetrics(metrics)

				m[id] = metaGen.Generate(NodeResource, r)

			case *kubernetes.Deployment:
				m[id] = metaGen.Generate(DeploymentResource, r)
			case *kubernetes.Job:
				m[id] = metaGen.Generate(JobResource, r)
			case *kubernetes.CronJob:
				m[id] = metaGen.Generate(CronJobResource, r)
			case *kubernetes.Service:
				m[id] = serviceMetaGen.Generate(r)
			case *kubernetes.StatefulSet:
				m[id] = metaGen.Generate(StatefulSetResource, r)
			case *kubernetes.Namespace:
				m[id] = metaGen.Generate(NamespaceResource, r)
			case *kubernetes.ReplicaSet:
				m[id] = metaGen.Generate(ReplicaSetResource, r)
			case *kubernetes.DaemonSet:
				m[id] = metaGen.Generate(DaemonSetResource, r)
			case *kubernetes.PersistentVolume:
				m[id] = metaGen.Generate(PersistentVolumeResource, r)
			case *kubernetes.PersistentVolumeClaim:
				m[id] = metaGen.Generate(PersistentVolumeClaimResource, r)
			case *kubernetes.StorageClass:
				m[id] = metaGen.Generate(StorageClassResource, r)
			default:
				m[id] = metaGen.Generate(r.GetObjectKind().GroupVersionKind().Kind, r)
			}
		},
		// delete
		func(m map[string]mapstr.M, r kubernetes.Resource) {
			accessor, _ := meta.Accessor(r)

			switch r := r.(type) {
			case *kubernetes.Node:
				nodeName := r.GetObjectMeta().GetName()
				metricsRepo.DeleteNodeStore(nodeName)
			}

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
	metricsRepo *MetricsRepo,
	nodeScope bool) Enricher {

	var replicaSetWatcher, jobWatcher kubernetes.Watcher
	config, err := GetValidatedConfig(base)
	if err != nil {
		logp.Info("Kubernetes metricset enriching is disabled")
		return &nilEnricher{}
	}

	client, err := kubernetes.GetKubernetesClient(config.KubeConfig, config.KubeClientOptions)
	if err != nil {
		logp.Err("Error creating Kubernetes client: %s", err)
		return &nilEnricher{}
	}

	watcher, nodeWatcher, namespaceWatcher := getResourceMetadataWatchers(config, &kubernetes.Pod{}, client, nodeScope)
	if watcher == nil {
		return &nilEnricher{}
	}

	// Resource is Pod so we need to create watchers for Replicasets and Jobs that it might belongs to
	// in order to be able to retrieve 2nd layer Owner metadata like in case of:
	// Deployment -> Replicaset -> Pod
	// CronJob -> job -> Pod
	if config.AddResourceMetadata.Deployment {
		replicaSetWatcher, err = kubernetes.NewNamedWatcher("resource_metadata_enricher_rs", client, &kubernetes.ReplicaSet{}, kubernetes.WatchOptions{
			SyncTimeout: config.SyncPeriod,
		}, nil)
		if err != nil {
			logp.Err("Error creating watcher for %T due to error %+v", &kubernetes.Namespace{}, err)
			return &nilEnricher{}
		}
	}
	if config.AddResourceMetadata.CronJob {
		jobWatcher, err = kubernetes.NewNamedWatcher("resource_metadata_enricher_job", client, &kubernetes.Job{}, kubernetes.WatchOptions{
			SyncTimeout: config.SyncPeriod,
		}, nil)
		if err != nil {
			logp.Err("Error creating watcher for %T due to error %+v", &kubernetes.Job{}, err)
			return &nilEnricher{}
		}
	}

	commonMetaConfig := metadata.Config{}
	if err := base.Module().UnpackConfig(&commonMetaConfig); err != nil {
		logp.Err("Error initializing Kubernetes metadata enricher: %s", err)
		return &nilEnricher{}
	}
	cfg, _ := conf.NewConfigFrom(&commonMetaConfig)

	metaGen := metadata.GetPodMetaGen(cfg, watcher, nodeWatcher, namespaceWatcher, replicaSetWatcher, jobWatcher, config.AddResourceMetadata)

	enricher := buildMetadataEnricher(watcher, nodeWatcher, namespaceWatcher, replicaSetWatcher, jobWatcher,
		// update
		func(m map[string]mapstr.M, r kubernetes.Resource) {
			pod, ok := r.(*kubernetes.Pod)
			if !ok {
				base.Logger().Debugf("Error while casting event: %s", ok)
			}
			pmeta := metaGen.Generate(pod)

			statuses := make(map[string]*kubernetes.PodContainerStatus)
			mapStatuses := func(s []kubernetes.PodContainerStatus) {
				for i := range s {
					statuses[s[i].Name] = &s[i]
				}
			}
			mapStatuses(pod.Status.ContainerStatuses)
			mapStatuses(pod.Status.InitContainerStatuses)

			nodeStore, _ := metricsRepo.AddNodeStore(pod.Spec.NodeName)
			podId := NewPodId(pod.Namespace, pod.Name)
			podStore, _ := nodeStore.AddPodStore(podId)

			for _, container := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
				cmeta := mapstr.M{}
				metrics := NewContainerMetrics()

				if cpu, ok := container.Resources.Limits["cpu"]; ok {
					if q, err := resource.ParseQuantity(cpu.String()); err == nil {
						metrics.CoresLimit = NewFloat64Metric(float64(q.MilliValue()) / 1000)
					}
				}
				if memory, ok := container.Resources.Limits["memory"]; ok {
					if q, err := resource.ParseQuantity(memory.String()); err == nil {
						metrics.MemoryLimit = NewFloat64Metric(float64(q.Value()))
					}
				}

				containerStore, _ := podStore.AddContainerStore(container.Name)
				containerStore.SetContainerMetrics(metrics)

				if s, ok := statuses[container.Name]; ok {
					// Extracting id and runtime ECS fields from ContainerID
					// which is in the form of <container.runtime>://<container.id>
					split := strings.Index(s.ContainerID, "://")
					if split != -1 {
						kubernetes2.ShouldPut(cmeta, "container.id", s.ContainerID[split+3:], base.Logger())

						kubernetes2.ShouldPut(cmeta, "container.runtime", s.ContainerID[:split], base.Logger())
					}
				}

				id := join(pod.GetObjectMeta().GetNamespace(), pod.GetObjectMeta().GetName(), container.Name)
				cmeta.DeepUpdate(pmeta)
				m[id] = cmeta
			}
		},
		// delete
		func(m map[string]mapstr.M, r kubernetes.Resource) {
			pod, ok := r.(*kubernetes.Pod)
			if !ok {
				base.Logger().Debugf("Error while casting event: %s", ok)
			}
			podId := NewPodId(pod.Namespace, pod.Name)
			nodeStore := metricsRepo.GetNodeStore(pod.Spec.NodeName)
			nodeStore.DeletePodStore(podId)

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

func getResourceMetadataWatchers(
	config *kubernetesConfig,
	resource kubernetes.Resource,
	client k8sclient.Interface, nodeScope bool) (kubernetes.Watcher, kubernetes.Watcher, kubernetes.Watcher) {

	var err error

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

func GetValidatedConfig(base mb.BaseMetricSet) (*kubernetesConfig, error) {
	config, err := GetConfig(base)
	if err != nil {
		logp.Err("Error while getting config: %v", err)
		return nil, err
	}

	config, err = validateConfig(config)
	if err != nil {
		logp.Err("Error while validating config: %v", err)
		return nil, err
	}
	return config, nil
}

func validateConfig(config *kubernetesConfig) (*kubernetesConfig, error) {
	if !config.AddMetadata {
		return nil, errors.New("metadata enriching is disabled")
	}
	return config, nil
}

func GetConfig(base mb.BaseMetricSet) (*kubernetesConfig, error) {
	config := &kubernetesConfig{
		AddMetadata:         true,
		SyncPeriod:          time.Minute * 10,
		AddResourceMetadata: metadata.GetDefaultResourceMetadataConfig(),
	}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, errors.New("error unpacking configs")
	}

	return config, nil
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
	replicasetWatcher kubernetes.Watcher,
	jobWatcher kubernetes.Watcher,
	update func(map[string]mapstr.M, kubernetes.Resource),
	delete func(map[string]mapstr.M, kubernetes.Resource),
	index func(e mapstr.M) string) *enricher {

	enricher := enricher{
		metadata:          map[string]mapstr.M{},
		index:             index,
		watcher:           watcher,
		nodeWatcher:       nodeWatcher,
		namespaceWatcher:  namespaceWatcher,
		replicasetWatcher: replicasetWatcher,
		jobWatcher:        jobWatcher,
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

		if m.replicasetWatcher != nil {
			if err := m.replicasetWatcher.Start(); err != nil {
				logp.Warn("Error starting replicaset watcher: %s", err)
			}
		}

		if m.jobWatcher != nil {
			if err := m.jobWatcher.Start(); err != nil {
				logp.Warn("Error starting job watcher: %s", err)
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

		if m.replicasetWatcher != nil {
			m.replicasetWatcher.Stop()
		}

		if m.jobWatcher != nil {
			m.jobWatcher.Stop()
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

func GetClusterECSMeta(cfg *conf.C, client k8sclient.Interface, logger *logp.Logger) (mapstr.M, error) {
	clusterInfo, err := metadata.GetKubernetesClusterIdentifier(cfg, client)
	if err != nil {
		return nil, fmt.Errorf("fail to get kubernetes cluster metadata: %w", err)
	}
	ecsClusterMeta := mapstr.M{}
	if clusterInfo.URL != "" {
		kubernetes2.ShouldPut(ecsClusterMeta, "orchestrator.cluster.url", clusterInfo.URL, logger)
	}
	if clusterInfo.Name != "" {
		kubernetes2.ShouldPut(ecsClusterMeta, "orchestrator.cluster.name", clusterInfo.Name, logger)
	}
	return ecsClusterMeta, nil
}

// AddClusterECSMeta adds ECS orchestrator fields
func AddClusterECSMeta(base mb.BaseMetricSet) mapstr.M {
	config, err := GetValidatedConfig(base)
	if err != nil {
		logp.Info("could not retrieve validated config")
		return mapstr.M{}
	}
	client, err := kubernetes.GetKubernetesClient(config.KubeConfig, config.KubeClientOptions)
	if err != nil {
		logp.Err("fail to get kubernetes client: %s", err)
		return mapstr.M{}
	}
	cfg, _ := conf.NewConfigFrom(&config)
	ecsClusterMeta, err := GetClusterECSMeta(cfg, client, base.Logger())
	if err != nil {
		logp.Info("could not retrieve cluster metadata: %s", err)
		return mapstr.M{}
	}
	return ecsClusterMeta
}
