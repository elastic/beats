// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kubernetes

import (
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/safemapstr"

	k8s "k8s.io/client-go/kubernetes"

	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes/metadata"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
)

type pod struct {
	logger           *logp.Logger
	cleanupTimeout   time.Duration
	comm             composable.DynamicProviderComm
	scope            string
	config           *Config
	metagen          metadata.MetaGen
	watcher          kubernetes.Watcher
	nodeWatcher      kubernetes.Watcher
	namespaceWatcher kubernetes.Watcher

	// Mutex used by configuration updates not triggered by the main watcher,
	// to avoid race conditions between cross updates and deletions.
	// Other updaters must use a write lock.
	crossUpdate sync.RWMutex
}

type providerData struct {
	uid        string
	mapping    map[string]interface{}
	processors []map[string]interface{}
}

// NewPodEventer creates an eventer that can discover and process pod objects
func NewPodEventer(
	comm composable.DynamicProviderComm,
	cfg *Config,
	logger *logp.Logger,
	client k8s.Interface,
	scope string) (Eventer, error) {
	watcher, err := kubernetes.NewNamedWatcher("agent-pod", client, &kubernetes.Pod{}, kubernetes.WatchOptions{
		SyncTimeout:  cfg.SyncPeriod,
		Node:         cfg.Node,
		Namespace:    cfg.Namespace,
		HonorReSyncs: true,
	}, nil)
	if err != nil {
		return nil, errors.New(err, "couldn't create kubernetes watcher")
	}

	options := kubernetes.WatchOptions{
		SyncTimeout: cfg.SyncPeriod,
		Node:        cfg.Node,
	}
	metaConf := cfg.AddResourceMetadata

	nodeWatcher, err := kubernetes.NewNamedWatcher("agent-node", client, &kubernetes.Node{}, options, nil)
	if err != nil {
		logger.Errorf("couldn't create watcher for %T due to error %+v", &kubernetes.Node{}, err)
	}
	namespaceWatcher, err := kubernetes.NewNamedWatcher("agent-namespace", client, &kubernetes.Namespace{}, kubernetes.WatchOptions{
		SyncTimeout: cfg.SyncPeriod,
	}, nil)
	if err != nil {
		logger.Errorf("couldn't create watcher for %T due to error %+v", &kubernetes.Namespace{}, err)
	}

	rawConfig, err := common.NewConfigFrom(cfg)
	if err != nil {
		return nil, errors.New(err, "failed to unpack configuration")
	}
	metaGen := metadata.GetPodMetaGen(rawConfig, watcher, nodeWatcher, namespaceWatcher, metaConf)

	p := &pod{
		logger:           logger,
		cleanupTimeout:   cfg.CleanupTimeout,
		comm:             comm,
		scope:            scope,
		config:           cfg,
		metagen:          metaGen,
		watcher:          watcher,
		nodeWatcher:      nodeWatcher,
		namespaceWatcher: namespaceWatcher,
	}

	watcher.AddEventHandler(p)

	if nodeWatcher != nil && metaConf.Node.Enabled() {
		updater := kubernetes.NewNodePodUpdater(p.unlockedUpdate, watcher.Store(), &p.crossUpdate)
		nodeWatcher.AddEventHandler(updater)
	}

	if namespaceWatcher != nil && metaConf.Namespace.Enabled() {
		updater := kubernetes.NewNamespacePodUpdater(p.unlockedUpdate, watcher.Store(), &p.crossUpdate)
		namespaceWatcher.AddEventHandler(updater)
	}

	return p, nil
}

// Start starts the eventer
func (p *pod) Start() error {
	if p.nodeWatcher != nil {
		err := p.nodeWatcher.Start()
		if err != nil {
			return err
		}
	}

	if p.namespaceWatcher != nil {
		if err := p.namespaceWatcher.Start(); err != nil {
			return err
		}
	}

	return p.watcher.Start()
}

// Stop stops the eventer
func (p *pod) Stop() {
	p.watcher.Stop()

	if p.namespaceWatcher != nil {
		p.namespaceWatcher.Stop()
	}

	if p.nodeWatcher != nil {
		p.nodeWatcher.Stop()
	}
}

func (p *pod) emitRunning(pod *kubernetes.Pod) {

	namespaceAnnotations := kubernetes.PodNamespaceAnnotations(pod, p.namespaceWatcher)

	data := generatePodData(pod, p.config, p.metagen, namespaceAnnotations)
	data.mapping["scope"] = p.scope
	// Emit the pod
	// We emit Pod + containers to ensure that configs matching Pod only
	// get Pod metadata (not specific to any container)
	p.comm.AddOrUpdate(data.uid, PodPriority, data.mapping, data.processors)

	// Emit all containers in the pod
	// TODO: deal with init containers stopping after initialization
	p.emitContainers(pod, namespaceAnnotations)
}

func (p *pod) emitContainers(pod *kubernetes.Pod, namespaceAnnotations common.MapStr) {
	generateContainerData(p.comm, pod, p.config, p.metagen, namespaceAnnotations)
}

func (p *pod) emitStopped(pod *kubernetes.Pod) {
	p.comm.Remove(string(pod.GetUID()))

	for _, c := range pod.Spec.Containers {
		// ID is the combination of pod UID + container name
		eventID := fmt.Sprintf("%s.%s", pod.GetObjectMeta().GetUID(), c.Name)
		p.comm.Remove(eventID)
	}

	for _, c := range pod.Spec.InitContainers {
		// ID is the combination of pod UID + container name
		eventID := fmt.Sprintf("%s.%s", pod.GetObjectMeta().GetUID(), c.Name)
		p.comm.Remove(eventID)
	}
}

// OnAdd ensures processing of pod objects that are newly added
func (p *pod) OnAdd(obj interface{}) {
	p.crossUpdate.RLock()
	defer p.crossUpdate.RUnlock()

	p.logger.Debugf("pod add: %+v", obj)
	p.emitRunning(obj.(*kubernetes.Pod))
}

// OnUpdate emits events for a given pod depending on the state of the pod,
// if it is terminating, a stop event is scheduled, if not, a stop and a start
// events are sent sequentially to recreate the resources assotiated to the pod.
func (p *pod) OnUpdate(obj interface{}) {
	p.crossUpdate.RLock()
	defer p.crossUpdate.RUnlock()

	p.unlockedUpdate(obj)
}

func (p *pod) unlockedUpdate(obj interface{}) {
	p.logger.Debugf("Watcher Pod update: %+v", obj)
	pod := obj.(*kubernetes.Pod)
	p.emitRunning(pod)
}

// OnDelete stops pod objects that are deleted
func (p *pod) OnDelete(obj interface{}) {
	p.crossUpdate.RLock()
	defer p.crossUpdate.RUnlock()

	p.logger.Debugf("pod delete: %+v", obj)
	pod := obj.(*kubernetes.Pod)
	time.AfterFunc(p.cleanupTimeout, func() {
		p.emitStopped(pod)
	})
}

func generatePodData(
	pod *kubernetes.Pod,
	cfg *Config,
	kubeMetaGen metadata.MetaGen,
	namespaceAnnotations common.MapStr) providerData {

	meta := kubeMetaGen.Generate(pod)
	kubemetaMap, err := meta.GetValue("kubernetes")
	if err != nil {
		return providerData{}
	}

	// k8sMapping includes only the metadata that fall under kubernetes.*
	// and these are available as dynamic vars through the provider
	k8sMapping := map[string]interface{}(kubemetaMap.(common.MapStr).Clone())

	if len(namespaceAnnotations) != 0 {
		k8sMapping["namespace_annotations"] = namespaceAnnotations
	}
	// Pass annotations to all events so that it can be used in templating and by annotation builders.
	annotations := common.MapStr{}
	for k, v := range pod.GetObjectMeta().GetAnnotations() {
		safemapstr.Put(annotations, k, v)
	}
	k8sMapping["annotations"] = annotations

	processors := []map[string]interface{}{}
	// meta map includes metadata that go under kubernetes.*
	// but also other ECS fields like orchestrator.*
	for field, metaMap := range meta {
		processor := map[string]interface{}{
			"add_fields": map[string]interface{}{
				"fields": metaMap,
				"target": field,
			},
		}
		processors = append(processors, processor)
	}

	return providerData{
		uid:        string(pod.GetUID()),
		mapping:    k8sMapping,
		processors: processors,
	}
}

func generateContainerData(
	comm composable.DynamicProviderComm,
	pod *kubernetes.Pod,
	cfg *Config,
	kubeMetaGen metadata.MetaGen,
	namespaceAnnotations common.MapStr) {

	containers := kubernetes.GetContainersInPod(pod)

	// Pass annotations to all events so that it can be used in templating and by annotation builders.
	annotations := common.MapStr{}
	for k, v := range pod.GetObjectMeta().GetAnnotations() {
		safemapstr.Put(annotations, k, v)
	}

	for _, c := range containers {
		// If it doesn't have an ID, container doesn't exist in
		// the runtime, emit only an event if we are stopping, so
		// we are sure of cleaning up configurations.
		if c.ID == "" {
			continue
		}

		// ID is the combination of pod UID + container name
		eventID := fmt.Sprintf("%s.%s", pod.GetObjectMeta().GetUID(), c.Spec.Name)

		meta := kubeMetaGen.Generate(pod, metadata.WithFields("container.name", c.Spec.Name))
		kubemetaMap, err := meta.GetValue("kubernetes")
		if err != nil {
			continue
		}

		// k8sMapping includes only the metadata that fall under kubernetes.*
		// and these are available as dynamic vars through the provider
		k8sMapping := map[string]interface{}(kubemetaMap.(common.MapStr).Clone())

		if len(namespaceAnnotations) != 0 {
			k8sMapping["namespace_annotations"] = namespaceAnnotations
		}
		// add annotations to be discoverable by templates
		k8sMapping["annotations"] = annotations

		//container ECS fields
		cmeta := common.MapStr{
			"id":      c.ID,
			"runtime": c.Runtime,
			"image": common.MapStr{
				"name": c.Spec.Image,
			},
		}

		processors := []map[string]interface{}{
			{
				"add_fields": map[string]interface{}{
					"fields": cmeta,
					"target": "container",
				},
			},
		}
		// meta map includes metadata that go under kubernetes.*
		// but also other ECS fields like orchestrator.*
		for field, metaMap := range meta {
			processor := map[string]interface{}{
				"add_fields": map[string]interface{}{
					"fields": metaMap,
					"target": field,
				},
			}
			processors = append(processors, processor)
		}

		// add container metadata under kubernetes.container.* to
		// make them available to dynamic var resolution
		containerMeta := common.MapStr{
			"id":      c.ID,
			"name":    c.Spec.Name,
			"image":   c.Spec.Image,
			"runtime": c.Runtime,
		}
		if len(c.Spec.Ports) > 0 {
			for _, port := range c.Spec.Ports {
				containerMeta.Put("port", fmt.Sprintf("%v", port.ContainerPort))
				containerMeta.Put("port_name", port.Name)
				k8sMapping["container"] = containerMeta
				comm.AddOrUpdate(eventID, ContainerPriority, k8sMapping, processors)
			}
		} else {
			k8sMapping["container"] = containerMeta
			comm.AddOrUpdate(eventID, ContainerPriority, k8sMapping, processors)
		}
	}
}
