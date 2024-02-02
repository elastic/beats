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

// Enricher takes Kubernetes events and enrich them with k8s metadata
type Enricher interface {
	// Start will start the Kubernetes watcher on the first call, does nothing on the rest
	// errors are logged as warning
	Start(*Watchers)

	// Stop will stop the Kubernetes watcher
	Stop(*Watchers)

	// Enrich the given list of events
	Enrich([]mapstr.M)
}

type enricher struct {
	sync.RWMutex
	metadata      map[string]mapstr.M
	index         func(mapstr.M) string
	metricsetName string
	resourceName  string
	isPod         bool
	config        *kubernetesConfig
	log           *logp.Logger
}

type nilEnricher struct{}

func (*nilEnricher) Start(*Watchers)   {}
func (*nilEnricher) Stop(*Watchers)    {}
func (*nilEnricher) Enrich([]mapstr.M) {}

type watcherData struct {
	// list of metricsets using this watcher
	// metricsets are used instead of resource names to avoid conflicts between
	// state_pod / pod, state_node / node, state_container / container
	metricsetsUsing []string

	watcher kubernetes.Watcher
	started bool // true if watcher has started, false otherwise

	enrichers []*enricher // list of enrichers using this watcher
}

type Watchers struct {
	watchersMap map[string]*watcherData
	lock        sync.RWMutex
}

const selector = "kubernetes"

const StateMetricsetPrefix = "state_"

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

func NewWatchers() *Watchers {
	watchers := &Watchers{
		watchersMap: make(map[string]*watcherData),
	}
	return watchers
}

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

// getExtraWatchers returns a list of the extra resources to watch based on some resource.
// The full list can be seen in https://github.com/elastic/beats/issues/37243, at Expected Watchers section.
func getExtraWatchers(resourceName string, addResourceMetadata *metadata.AddResourceMetadataConfig) []string {
	switch resourceName {
	case PodResource:
		extra := []string{NamespaceResource, NodeResource}
		// We need to create watchers for ReplicaSets and Jobs that it might belong to,
		// in order to be able to retrieve 2nd layer Owner metadata like in case of:
		// Deployment -> Replicaset -> Pod
		// CronJob -> job -> Pod
		if addResourceMetadata != nil && addResourceMetadata.Deployment {
			extra = append(extra, ReplicaSetResource)
		}
		if addResourceMetadata != nil && addResourceMetadata.CronJob {
			extra = append(extra, JobResource)
		}
		return extra
	case ServiceResource:
		return []string{NamespaceResource}
	case DeploymentResource:
		return []string{NamespaceResource}
	case ReplicaSetResource:
		return []string{NamespaceResource}
	case StatefulSetResource:
		return []string{NamespaceResource}
	case DaemonSetResource:
		return []string{NamespaceResource}
	case JobResource:
		return []string{NamespaceResource}
	case CronJobResource:
		return []string{NamespaceResource}
	case PersistentVolumeResource:
		return []string{}
	case PersistentVolumeClaimResource:
		return []string{NamespaceResource}
	case StorageClassResource:
		return []string{}
	case NodeResource:
		return []string{}
	case NamespaceResource:
		return []string{}
	default:
		return []string{}
	}
}

// getResourceName returns the name of the resource for a metricset
// Example: state_pod metricset uses pod resource
// Exception is state_namespace
func getResourceName(metricsetName string) string {
	resourceName := metricsetName
	if resourceName != NamespaceResource {
		resourceName = strings.ReplaceAll(resourceName, StateMetricsetPrefix, "")
	}
	return resourceName
}

// getWatchOptions builds the kubernetes.WatchOptions{} needed for the watcher based on the config and nodeScope
func getWatchOptions(config *kubernetesConfig, nodeScope bool, client k8sclient.Interface, log *logp.Logger) (*kubernetes.WatchOptions, error) {
	var err error
	options := kubernetes.WatchOptions{
		SyncTimeout: config.SyncPeriod,
	}

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
			return nil, fmt.Errorf("couldn't discover kubernetes node: %w", err)
		}
	}
	return &options, err
}

func isNamespaced(resourceName string) bool {
	if resourceName == NodeResource || resourceName == PersistentVolumeResource || resourceName == StorageClassResource ||
		resourceName == NamespaceResource {
		return false
	}
	return true
}

// createWatcher creates a watcher for a specific resource
func createWatcher(
	resourceName string,
	resource kubernetes.Resource,
	options kubernetes.WatchOptions,
	client k8sclient.Interface,
	resourceWatchers *Watchers,
	namespace string) (bool, error) {

	resourceWatchers.lock.Lock()
	defer resourceWatchers.lock.Unlock()

	_, ok := resourceWatchers.watchersMap[resourceName]
	// if it does not exist, create the watcher
	if !ok {
		// check if we need to add namespace to the options
		if isNamespaced(resourceName) {
			options.Namespace = namespace
		}
		watcher, err := kubernetes.NewNamedWatcher(resourceName, client, resource, options, nil)
		if err != nil {
			return false, err
		}
		resourceWatchers.watchersMap[resourceName] = &watcherData{watcher: watcher, started: false}
		return true, nil
	}
	return false, nil
}

// addToMetricsetsUsing adds metricset identified by metricsetUsing to the list of resources using the watcher
// identified by resourceName
func addToMetricsetsUsing(resourceName string, metricsetUsing string, resourceWatchers *Watchers) {
	resourceWatchers.lock.Lock()
	defer resourceWatchers.lock.Unlock()

	data, ok := resourceWatchers.watchersMap[resourceName]
	if ok {
		contains := false
		for _, which := range data.metricsetsUsing {
			if which == metricsetUsing {
				contains = true
				break
			}
		}
		// add this resource to the list of resources using it
		if !contains {
			data.metricsetsUsing = append(data.metricsetsUsing, metricsetUsing)
		}
	}
}

// removeFromMetricsetsUsing returns true if element was removed and new size of array.
// The cache should be locked when called.
func removeFromMetricsetsUsing(resourceName string, notUsingName string, resourceWatchers *Watchers) (bool, int) {
	data, ok := resourceWatchers.watchersMap[resourceName]
	removed := false
	if ok {
		newIndex := 0
		for i, which := range data.metricsetsUsing {
			if which == notUsingName {
				removed = true
			} else {
				data.metricsetsUsing[newIndex] = data.metricsetsUsing[i]
				newIndex++
			}
		}
		data.metricsetsUsing = data.metricsetsUsing[:newIndex]
		return removed, len(data.metricsetsUsing)
	}
	return removed, 0
}

// createAllWatchers creates all the watchers required by a metricset
func createAllWatchers(
	client k8sclient.Interface,
	metricsetName string,
	resourceName string,
	nodeScope bool,
	config *kubernetesConfig,
	log *logp.Logger,
	resourceWatchers *Watchers,
) error {
	res := getResource(resourceName)
	if res == nil {
		return fmt.Errorf("resource for name %s does not exist. Watcher cannot be created", resourceName)
	}

	options, err := getWatchOptions(config, nodeScope, client, log)
	if err != nil {
		return err
	}

	// Create a watcher for the given resource.
	// If it fails, we return an error, so we can stop the extra watchers from creating.
	created, err := createWatcher(resourceName, res, *options, client, resourceWatchers, config.Namespace)
	if err != nil {
		return fmt.Errorf("error initializing Kubernetes watcher %s, required by %s: %w", resourceName, metricsetName, err)
	} else if created {
		log.Debugf("Created watcher %s successfully, created by %s.", resourceName, metricsetName)
	}
	addToMetricsetsUsing(resourceName, metricsetName, resourceWatchers)

	// Create the extra watchers required by this resource
	extraWatchers := getExtraWatchers(resourceName, config.AddResourceMetadata)
	for _, extra := range extraWatchers {
		extraRes := getResource(extra)
		if extraRes != nil {
			created, err = createWatcher(extra, extraRes, *options, client, resourceWatchers, config.Namespace)
			if err != nil {
				log.Errorf("Error initializing Kubernetes watcher %s, required by %s: %s", extra, metricsetName, err)
			} else {
				if created {
					log.Debugf("Created watcher %s successfully, created by %s.", extra, metricsetName)
				}
				// add this metricset to the ones using the extra resource
				addToMetricsetsUsing(extra, metricsetName, resourceWatchers)
			}
		} else {
			log.Errorf("Resource for name %s does not exist. Watcher cannot be created.", extra)
		}
	}

	return nil
}

// createMetadataGen creates the metadata generator for resources in general
func createMetadataGen(client k8sclient.Interface, commonConfig *conf.C, addResourceMetadata *metadata.AddResourceMetadataConfig,
	resourceName string, resourceWatchers *Watchers) (*metadata.Resource, error) {

	resourceWatchers.lock.RLock()
	defer resourceWatchers.lock.RUnlock()

	resourceWatcher := resourceWatchers.watchersMap[resourceName]
	// This should not be possible since the watchers should have been created before
	if resourceWatcher == nil {
		return nil, fmt.Errorf("could not create the metadata generator, as the watcher for %s does not exist", resourceName)
	}

	var metaGen *metadata.Resource

	namespaceWatcher := resourceWatchers.watchersMap[NamespaceResource]
	if namespaceWatcher != nil {
		n := metadata.NewNamespaceMetadataGenerator(addResourceMetadata.Namespace,
			(*namespaceWatcher).watcher.Store(), client)
		metaGen = metadata.NewNamespaceAwareResourceMetadataGenerator(commonConfig, client, n)
	} else {
		metaGen = metadata.NewResourceMetadataGenerator(commonConfig, client)
	}

	return metaGen, nil
}

// createMetadataGenSpecific creates the metadata generator for a specific resource - pod or service
func createMetadataGenSpecific(client k8sclient.Interface, commonConfig *conf.C, addResourceMetadata *metadata.AddResourceMetadataConfig,
	resourceName string, resourceWatchers *Watchers) (metadata.MetaGen, error) {

	resourceWatchers.lock.RLock()
	defer resourceWatchers.lock.RUnlock()

	// The watcher for the resource needs to exist
	resWatcher := resourceWatchers.watchersMap[resourceName]
	if resWatcher == nil {
		return nil, fmt.Errorf("could not create the metadata generator, as the watcher for %s does not exist", resourceName)
	}

	var metaGen metadata.MetaGen
	if resourceName == PodResource {
		var nodeWatcher kubernetes.Watcher
		if watcher := resourceWatchers.watchersMap[NodeResource]; watcher != nil {
			nodeWatcher = (*watcher).watcher
		}
		var namespaceWatcher kubernetes.Watcher
		if watcher := resourceWatchers.watchersMap[NamespaceResource]; watcher != nil {
			namespaceWatcher = (*watcher).watcher
		}
		var replicaSetWatcher kubernetes.Watcher
		if watcher := resourceWatchers.watchersMap[ReplicaSetResource]; watcher != nil {
			replicaSetWatcher = (*watcher).watcher
		}
		var jobWatcher kubernetes.Watcher
		if watcher := resourceWatchers.watchersMap[JobResource]; watcher != nil {
			jobWatcher = (*watcher).watcher
		}

		metaGen = metadata.GetPodMetaGen(commonConfig, (*resWatcher).watcher, nodeWatcher, namespaceWatcher, replicaSetWatcher,
			jobWatcher, addResourceMetadata)
		return metaGen, nil
	} else if resourceName == ServiceResource {
		namespaceWatcher := resourceWatchers.watchersMap[NamespaceResource]
		if namespaceWatcher == nil {
			return nil, fmt.Errorf("could not create the metadata generator, as the watcher for namespace does not exist")
		}
		namespaceMeta := metadata.NewNamespaceMetadataGenerator(addResourceMetadata.Namespace,
			(*namespaceWatcher).watcher.Store(), client)
		metaGen = metadata.NewServiceMetadataGenerator(commonConfig, (*resWatcher).watcher.Store(),
			namespaceMeta, client)
		return metaGen, nil
	}

	// Should never reach this part, as this function is only for service or pod resources
	return metaGen, fmt.Errorf("failed to create a metadata generator for resource %s", resourceName)
}

func NewResourceMetadataEnricher(
	base mb.BaseMetricSet,
	metricsRepo *MetricsRepo,
	resourceWatchers *Watchers,
	nodeScope bool) Enricher {
	log := logp.NewLogger(selector)

	config, err := GetValidatedConfig(base)
	if err != nil {
		log.Info("Kubernetes metricset enriching is disabled")
		return &nilEnricher{}
	}

	// This type of config is needed for the metadata generator
	commonMetaConfig := metadata.Config{}
	if err := base.Module().UnpackConfig(&commonMetaConfig); err != nil {
		log.Errorf("Error initializing Kubernetes metadata enricher: %s", err)
		return &nilEnricher{}
	}
	commonConfig, _ := conf.NewConfigFrom(&commonMetaConfig)

	client, err := kubernetes.GetKubernetesClient(config.KubeConfig, config.KubeClientOptions)
	if err != nil {
		log.Errorf("Error creating Kubernetes client: %s", err)
		return &nilEnricher{}
	}

	metricsetName := base.Name()
	resourceName := getResourceName(metricsetName)

	err = createAllWatchers(client, metricsetName, resourceName, nodeScope, config, log, resourceWatchers)
	if err != nil {
		log.Errorf("Error starting the watchers: %s", err)
		return &nilEnricher{}
	}

	var specificMetaGen metadata.MetaGen
	var generalMetaGen *metadata.Resource
	if resourceName == ServiceResource || resourceName == PodResource {
		specificMetaGen, err = createMetadataGenSpecific(client, commonConfig, config.AddResourceMetadata, resourceName, resourceWatchers)
	} else {
		generalMetaGen, err = createMetadataGen(client, commonConfig, config.AddResourceMetadata, resourceName, resourceWatchers)
	}
	if err != nil {
		log.Errorf("Error trying to create the metadata generators: %s", err)
		return &nilEnricher{}
	}

	updateFunc := func(m map[string]mapstr.M, r kubernetes.Resource) {
		accessor, _ := meta.Accessor(r)
		id := join(accessor.GetNamespace(), accessor.GetName())

		switch r := r.(type) {
		case *kubernetes.Pod:
			m[id] = specificMetaGen.Generate(r)

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

			m[id] = generalMetaGen.Generate(NodeResource, r)
		case *kubernetes.Deployment:
			m[id] = generalMetaGen.Generate(DeploymentResource, r)
		case *kubernetes.Job:
			m[id] = generalMetaGen.Generate(JobResource, r)
		case *kubernetes.CronJob:
			m[id] = generalMetaGen.Generate(CronJobResource, r)
		case *kubernetes.Service:
			m[id] = specificMetaGen.Generate(r)
		case *kubernetes.StatefulSet:
			m[id] = generalMetaGen.Generate(StatefulSetResource, r)
		case *kubernetes.Namespace:
			m[id] = generalMetaGen.Generate(NamespaceResource, r)
		case *kubernetes.ReplicaSet:
			m[id] = generalMetaGen.Generate(ReplicaSetResource, r)
		case *kubernetes.DaemonSet:
			m[id] = generalMetaGen.Generate(DaemonSetResource, r)
		case *kubernetes.PersistentVolume:
			m[id] = generalMetaGen.Generate(PersistentVolumeResource, r)
		case *kubernetes.PersistentVolumeClaim:
			m[id] = generalMetaGen.Generate(PersistentVolumeClaimResource, r)
		case *kubernetes.StorageClass:
			m[id] = generalMetaGen.Generate(StorageClassResource, r)
		default:
			m[id] = generalMetaGen.Generate(r.GetObjectKind().GroupVersionKind().Kind, r)
		}
	}

	deleteFunc := func(m map[string]mapstr.M, r kubernetes.Resource) {
		accessor, _ := meta.Accessor(r)

		switch r := r.(type) {
		case *kubernetes.Node:
			nodeName := r.GetObjectMeta().GetName()
			metricsRepo.DeleteNodeStore(nodeName)
		}

		id := join(accessor.GetNamespace(), accessor.GetName())
		delete(m, id)
	}

	indexFunc := func(e mapstr.M) string {
		return join(getString(e, mb.ModuleDataKey+".namespace"), getString(e, "name"))
	}

	enricher := buildMetadataEnricher(metricsetName, resourceName, resourceWatchers, config, updateFunc, deleteFunc, indexFunc, log)
	if resourceName == PodResource {
		enricher.isPod = true
	}

	return enricher
}

// NewContainerMetadataEnricher returns an Enricher configured for container events
func NewContainerMetadataEnricher(
	base mb.BaseMetricSet,
	metricsRepo *MetricsRepo,
	resourceWatchers *Watchers,
	nodeScope bool) Enricher {

	log := logp.NewLogger(selector)

	config, err := GetValidatedConfig(base)
	if err != nil {
		log.Info("Kubernetes metricset enriching is disabled")
		return &nilEnricher{}
	}

	// This type of config is needed for the metadata generator
	commonMetaConfig := metadata.Config{}
	if err := base.Module().UnpackConfig(&commonMetaConfig); err != nil {
		log.Errorf("Error initializing Kubernetes metadata enricher: %s", err)
		return &nilEnricher{}
	}
	commonConfig, _ := conf.NewConfigFrom(&commonMetaConfig)

	client, err := kubernetes.GetKubernetesClient(config.KubeConfig, config.KubeClientOptions)
	if err != nil {
		log.Errorf("Error creating Kubernetes client: %s", err)
		return &nilEnricher{}
	}

	metricsetName := base.Name()

	err = createAllWatchers(client, metricsetName, PodResource, nodeScope, config, log, resourceWatchers)
	if err != nil {
		log.Errorf("Error starting the watchers: %s", err)
		return &nilEnricher{}
	}

	metaGen, err := createMetadataGenSpecific(client, commonConfig, config.AddResourceMetadata, PodResource, resourceWatchers)
	if err != nil {
		log.Errorf("Error trying to create the metadata generators: %s", err)
		return &nilEnricher{}
	}

	updateFunc := func(m map[string]mapstr.M, r kubernetes.Resource) {
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
	}

	deleteFunc := func(m map[string]mapstr.M, r kubernetes.Resource) {
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
	}

	indexFunc := func(e mapstr.M) string {
		return join(getString(e, mb.ModuleDataKey+".namespace"), getString(e, mb.ModuleDataKey+".pod.name"), getString(e, "name"))
	}

	enricher := buildMetadataEnricher(metricsetName, PodResource, resourceWatchers, config, updateFunc, deleteFunc, indexFunc, log)

	return enricher
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
	metricsetName string,
	resourceName string,
	resourceWatchers *Watchers,
	config *kubernetesConfig,
	update func(map[string]mapstr.M, kubernetes.Resource),
	delete func(map[string]mapstr.M, kubernetes.Resource),
	index func(e mapstr.M) string,
	log *logp.Logger) *enricher {

	enricher := &enricher{
		metadata:      map[string]mapstr.M{},
		index:         index,
		resourceName:  resourceName,
		metricsetName: metricsetName,
		config:        config,
		log:           log,
	}

	resourceWatchers.lock.Lock()
	defer resourceWatchers.lock.Unlock()

	watcher := resourceWatchers.watchersMap[resourceName]
	if watcher != nil {
		watcher.enrichers = append(watcher.enrichers, enricher)
		watcher.watcher.AddEventHandler(kubernetes.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				for _, enricher := range watcher.enrichers {
					enricher.Lock()
					update(enricher.metadata, obj.(kubernetes.Resource))
					enricher.Unlock()
				}
			},
			UpdateFunc: func(obj interface{}) {
				for _, enricher := range watcher.enrichers {
					enricher.Lock()
					update(enricher.metadata, obj.(kubernetes.Resource))
					enricher.Unlock()
				}
			},
			DeleteFunc: func(obj interface{}) {
				for _, enricher := range watcher.enrichers {
					enricher.Lock()
					delete(enricher.metadata, obj.(kubernetes.Resource))
					enricher.Unlock()
				}
			},
		})
	}

	return enricher
}

func (e *enricher) Start(resourceWatchers *Watchers) {
	resourceWatchers.lock.Lock()
	defer resourceWatchers.lock.Unlock()

	resourceWatcher := resourceWatchers.watchersMap[e.resourceName]
	if resourceWatcher != nil && !resourceWatcher.started {
		if err := resourceWatcher.watcher.Start(); err != nil {
			e.log.Warnf("Error starting %s watcher: %s", e.resourceName, err)
		} else {
			resourceWatcher.started = true
		}
	}

	extras := getExtraWatchers(e.resourceName, e.config.AddResourceMetadata)
	for _, extra := range extras {
		extraWatcher := resourceWatchers.watchersMap[extra]
		if extraWatcher != nil && !extraWatcher.started {
			if err := extraWatcher.watcher.Start(); err != nil {
				e.log.Warnf("Error starting %s watcher: %s", extra, err)
			} else {
				extraWatcher.started = true
			}
		}
	}
}

func (e *enricher) Stop(resourceWatchers *Watchers) {
	resourceWatchers.lock.Lock()
	defer resourceWatchers.lock.Unlock()

	resourceWatcher := resourceWatchers.watchersMap[e.resourceName]
	if resourceWatcher != nil && resourceWatcher.started {
		_, size := removeFromMetricsetsUsing(e.resourceName, e.metricsetName, resourceWatchers)
		if size == 0 {
			resourceWatcher.watcher.Stop()
			resourceWatcher.started = false
		}
	}

	extras := getExtraWatchers(e.resourceName, e.config.AddResourceMetadata)
	for _, extra := range extras {
		extraWatcher := resourceWatchers.watchersMap[extra]
		if extraWatcher != nil && extraWatcher.started {
			_, size := removeFromMetricsetsUsing(extra, e.metricsetName, resourceWatchers)
			if size == 0 {
				extraWatcher.watcher.Stop()
				extraWatcher.started = false
			}
		}
	}
}

func (e *enricher) Enrich(events []mapstr.M) {
	e.RLock()
	defer e.RUnlock()

	for _, event := range events {
		if meta := e.metadata[e.index(event)]; meta != nil {
			k8s, err := meta.GetValue("kubernetes")
			if err != nil {
				continue
			}
			k8sMeta, ok := k8s.(mapstr.M)
			if !ok {
				continue
			}

			if e.isPod {
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
