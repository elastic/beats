package util

import (
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/kubernetes"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
)

// Enricher takes Kubernetes events and enrich them with k8s metadata
type Enricher interface {
	// Start will start the Kubernetes watcher on the first call, does nothing on the rest
	// errors are logged as warning
	Start()

	// Stop will stop the Kubernetes watcher
	Stop()

	// Enrich the given list of events
	Enrich([]common.MapStr)
}

type kubernetesConfig struct {
	// AddMetadata enables enriching metricset events with metadata from the API server
	AddMetadata bool          `config:"add_metadata"`
	InCluster   bool          `config:"in_cluster"`
	KubeConfig  string        `config:"kube_config"`
	Host        string        `config:"host"`
	SyncPeriod  time.Duration `config:"sync_period"`
}

type enricher struct {
	sync.RWMutex
	metadata       map[string]common.MapStr
	index          func(common.MapStr) string
	watcher        kubernetes.Watcher
	watcherStarted bool
}

// GetWatcher initializes a kubernetes watcher with the given
// scope (node or cluster), and resource type
func GetWatcher(base mb.BaseMetricSet, resource kubernetes.Resource, nodeScope bool) (kubernetes.Watcher, error) {
	config := kubernetesConfig{
		AddMetadata: true,
		InCluster:   true,
	}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	// Return nil if metadata enriching is disabled:
	if !config.AddMetadata {
		return nil, nil
	}

	client, err := kubernetes.GetKubernetesClient(config.InCluster, config.KubeConfig)
	if err != nil {
		return nil, err
	}

	options := kubernetes.WatchOptions{
		SyncTimeout: config.SyncPeriod,
	}

	// Watch objects in the node only
	if nodeScope {
		options.Node = kubernetes.DiscoverKubernetesNode(config.Host, config.InCluster, client)
	}

	return kubernetes.NewWatcher(client, resource, options)
}

// NewResourceMetadataEnricher returns an Enricher configured for kubernetes resource events
func NewResourceMetadataEnricher(
	base mb.BaseMetricSet,
	resource kubernetes.Resource,
	nodeScope bool) Enricher {

	watcher, err := GetWatcher(base, resource, nodeScope)
	if err != nil {
		logp.Warn("Error initializing Kubernetes metadata enricher: %s", err)
		return &nilEnricher{}
	}

	if watcher == nil {
		logp.Info("Kubernetes metricset enriching is disabled")
		return &nilEnricher{}
	}

	metaConfig := kubernetes.MetaGeneratorConfig{}
	if err := base.Module().UnpackConfig(&metaConfig); err != nil {
		logp.Warn("Error initializing Kubernetes metadata enricher: %s", err)
		return &nilEnricher{}
	}

	metaGen := kubernetes.NewMetaGeneratorFromConfig(&metaConfig)
	enricher := buildMetadataEnricher(watcher,
		// update
		func(m map[string]common.MapStr, r kubernetes.Resource) {
			id := join(r.GetMetadata().GetNamespace(), r.GetMetadata().GetName())
			m[id] = metaGen.ResourceMetadata(r)
		},
		// delete
		func(m map[string]common.MapStr, r kubernetes.Resource) {
			id := join(r.GetMetadata().GetNamespace(), r.GetMetadata().GetName())
			delete(m, id)
		},
		// index
		func(e common.MapStr) string {
			return join(getString(e, mb.ModuleDataKey+".namespace"), getString(e, "name"))
		},
	)

	return enricher
}

// NewContainerMetadataEnricher returns an Enricher configured for container events
func NewContainerMetadataEnricher(
	base mb.BaseMetricSet,
	nodeScope bool) Enricher {

	watcher, err := GetWatcher(base, &kubernetes.Pod{}, nodeScope)
	if err != nil {
		logp.Warn("Error initializing Kubernetes metadata enricher: %s", err)
		return &nilEnricher{}
	}

	if watcher == nil {
		logp.Info("Kubernetes metricset enriching is disabled")
		return &nilEnricher{}
	}

	metaConfig := kubernetes.MetaGeneratorConfig{}
	if err := base.Module().UnpackConfig(&metaConfig); err != nil {
		logp.Warn("Error initializing Kubernetes metadata enricher: %s", err)
		return &nilEnricher{}
	}

	metaGen := kubernetes.NewMetaGeneratorFromConfig(&metaConfig)
	enricher := buildMetadataEnricher(watcher,
		// update
		func(m map[string]common.MapStr, r kubernetes.Resource) {
			pod := r.(*kubernetes.Pod)
			meta := metaGen.ResourceMetadata(r)
			for _, container := range append(pod.GetSpec().GetContainers(), pod.GetSpec().GetInitContainers()...) {
				id := join(r.GetMetadata().GetNamespace(), r.GetMetadata().GetName(), container.GetName())
				m[id] = meta
			}
		},
		// delete
		func(m map[string]common.MapStr, r kubernetes.Resource) {
			pod := r.(*kubernetes.Pod)
			for _, container := range append(pod.GetSpec().GetContainers(), pod.GetSpec().GetInitContainers()...) {
				id := join(r.GetMetadata().GetNamespace(), r.GetMetadata().GetName(), container.GetName())
				delete(m, id)
			}
		},
		// index
		func(e common.MapStr) string {
			return join(getString(e, mb.ModuleDataKey+".namespace"), getString(e, mb.ModuleDataKey+".pod.name"), getString(e, "name"))
		},
	)

	return enricher
}

func getString(m common.MapStr, key string) string {
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
	update func(map[string]common.MapStr, kubernetes.Resource),
	delete func(map[string]common.MapStr, kubernetes.Resource),
	index func(e common.MapStr) string) Enricher {

	enricher := enricher{
		metadata: map[string]common.MapStr{},
		index:    index,
		watcher:  watcher,
	}

	watcher.AddEventHandler(kubernetes.ResourceEventHandlerFuncs{
		AddFunc: func(obj kubernetes.Resource) {
			enricher.Lock()
			defer enricher.Unlock()
			update(enricher.metadata, obj)
		},
		UpdateFunc: func(obj kubernetes.Resource) {
			enricher.Lock()
			defer enricher.Unlock()
			update(enricher.metadata, obj)
		},
		DeleteFunc: func(obj kubernetes.Resource) {
			enricher.Lock()
			defer enricher.Unlock()
			delete(enricher.metadata, obj)
		},
	})

	return &enricher
}

func (m *enricher) Start() {
	if !m.watcherStarted {
		err := m.watcher.Start()
		if err != nil {
			logp.Warn("Error starting Kubernetes watcher: %s", err)
		}
		m.watcherStarted = true
	}
}

func (m *enricher) Stop() {
	if m.watcherStarted {
		m.watcher.Stop()
		m.watcherStarted = false
	}
}

func (m *enricher) Enrich(events []common.MapStr) {
	for _, event := range events {
		if meta := m.metadata[m.index(event)]; meta != nil {
			event.DeepUpdate(common.MapStr{
				mb.ModuleDataKey: meta,
			})
		}
	}
}

type nilEnricher struct{}

func (*nilEnricher) Start()                 {}
func (*nilEnricher) Stop()                  {}
func (*nilEnricher) Enrich([]common.MapStr) {}
