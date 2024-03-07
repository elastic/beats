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

//go:build !aix

package kubernetes

import (
	"fmt"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	k8s "k8s.io/client-go/kubernetes"

	"github.com/elastic/elastic-agent-autodiscover/bus"
	"github.com/elastic/elastic-agent-autodiscover/kubernetes"
	"github.com/elastic/elastic-agent-autodiscover/kubernetes/metadata"
	"github.com/elastic/elastic-agent-autodiscover/utils"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type pod struct {
	uuid              uuid.UUID
	config            *Config
	metagen           metadata.MetaGen
	logger            *logp.Logger
	publishFunc       func([]bus.Event)
	watcher           kubernetes.Watcher
	nodeWatcher       kubernetes.Watcher
	namespaceWatcher  kubernetes.Watcher
	replicasetWatcher kubernetes.Watcher
	jobWatcher        kubernetes.Watcher

	// Mutex used by configuration updates not triggered by the main watcher,
	// to avoid race conditions between cross updates and deletions.
	// Other updaters must use a write lock.
	crossUpdate sync.RWMutex
}

// NewPodEventer creates an eventer that can discover and process pod objects
func NewPodEventer(uuid uuid.UUID, cfg *conf.C, client k8s.Interface, publish func(event []bus.Event)) (Eventer, error) {
	logger := logp.NewLogger("autodiscover.pod")

	var replicaSetWatcher, jobWatcher kubernetes.Watcher

	config := defaultConfig()
	err := cfg.Unpack(&config)
	if err != nil {
		return nil, err
	}

	// Ensure that node is set correctly whenever the scope is set to "node". Make sure that node is empty
	// when cluster scope is enforced.
	if config.Scope == "node" {
		nd := &kubernetes.DiscoverKubernetesNodeParams{
			ConfigHost:  config.Node,
			Client:      client,
			IsInCluster: kubernetes.IsInCluster(config.KubeConfig),
			HostUtils:   &kubernetes.DefaultDiscoveryUtils{},
		}
		config.Node, err = kubernetes.DiscoverKubernetesNode(logger, nd)
		if err != nil {
			return nil, fmt.Errorf("couldn't discover kubernetes node due to error %w", err)
		}
	} else {
		config.Node = ""
	}

	logger.Debugf("Initializing a new Kubernetes watcher using node: %v", config.Node)

	watcher, err := kubernetes.NewNamedWatcher("pod", client, &kubernetes.Pod{}, kubernetes.WatchOptions{
		SyncTimeout:  config.SyncPeriod,
		Node:         config.Node,
		Namespace:    config.Namespace,
		HonorReSyncs: true,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("couldn't create watcher for %T due to error %w", &kubernetes.Pod{}, err)
	}

	options := kubernetes.WatchOptions{
		SyncTimeout: config.SyncPeriod,
		Node:        config.Node,
		Namespace:   config.Namespace,
	}

	metaConf := config.AddResourceMetadata
	nodeWatcher, err := kubernetes.NewNamedWatcher("node", client, &kubernetes.Node{}, options, nil)
	if err != nil {
		logger.Errorf("couldn't create watcher for %T due to error %+v", &kubernetes.Node{}, err)
	}
	namespaceWatcher, err := kubernetes.NewNamedWatcher("namespace", client, &kubernetes.Namespace{}, kubernetes.WatchOptions{
		SyncTimeout: config.SyncPeriod,
	}, nil)
	if err != nil {
		logger.Errorf("couldn't create watcher for %T due to error %+v", &kubernetes.Namespace{}, err)
	}

	// Resource is Pod so we need to create watchers for Replicasets and Jobs that it might belongs to
	// in order to be able to retrieve 2nd layer Owner metadata like in case of:
	// Deployment -> Replicaset -> Pod
	// CronJob -> job -> Pod
	if metaConf.Deployment {
		replicaSetWatcher, err = kubernetes.NewNamedWatcher("resource_metadata_enricher_rs", client, &kubernetes.ReplicaSet{}, kubernetes.WatchOptions{
			SyncTimeout: config.SyncPeriod,
		}, nil)
		if err != nil {
			logger.Errorf("Error creating watcher for %T due to error %+v", &kubernetes.ReplicaSet{}, err)
		}
	}
	if metaConf.CronJob {
		jobWatcher, err = kubernetes.NewNamedWatcher("resource_metadata_enricher_job", client, &kubernetes.Job{}, kubernetes.WatchOptions{
			SyncTimeout: config.SyncPeriod,
		}, nil)
		if err != nil {
			logger.Errorf("Error creating watcher for %T due to error %+v", &kubernetes.Job{}, err)
		}
	}

	metaGen := metadata.GetPodMetaGen(cfg, watcher, nodeWatcher, namespaceWatcher, replicaSetWatcher, jobWatcher, metaConf)

	p := &pod{
		config:            config,
		uuid:              uuid,
		publishFunc:       publish,
		metagen:           metaGen,
		logger:            logger,
		watcher:           watcher,
		nodeWatcher:       nodeWatcher,
		namespaceWatcher:  namespaceWatcher,
		replicasetWatcher: replicaSetWatcher,
		jobWatcher:        jobWatcher,
	}

	watcher.AddEventHandler(p)

	if nodeWatcher != nil && (config.Hints.Enabled() || metaConf.Node.Enabled()) {
		updater := kubernetes.NewNodePodUpdater(p.unlockedUpdate, watcher.Store(), p.nodeWatcher, &p.crossUpdate)
		nodeWatcher.AddEventHandler(updater)
	}

	if namespaceWatcher != nil && (config.Hints.Enabled() || metaConf.Namespace.Enabled()) {
		updater := kubernetes.NewNamespacePodUpdater(p.unlockedUpdate, watcher.Store(), p.namespaceWatcher, &p.crossUpdate)
		namespaceWatcher.AddEventHandler(updater)
	}

	return p, nil
}

// OnAdd ensures processing of pod objects that are newly added.
func (p *pod) OnAdd(obj interface{}) {
	p.crossUpdate.RLock()
	defer p.crossUpdate.RUnlock()

	p.logger.Debugf("Watcher Pod add: %+v", obj)
	p.emit(obj.(*kubernetes.Pod), "start")
}

// OnUpdate handles events for pods that have been updated.
func (p *pod) OnUpdate(obj interface{}) {
	p.crossUpdate.RLock()
	defer p.crossUpdate.RUnlock()

	p.unlockedUpdate(obj)
}

func (p *pod) unlockedUpdate(obj interface{}) {
	p.logger.Debugf("Watcher Pod update: %+v", obj)
	p.emit(obj.(*kubernetes.Pod), "stop")
	p.emit(obj.(*kubernetes.Pod), "start")
}

// OnDelete stops pod objects that are deleted.
func (p *pod) OnDelete(obj interface{}) {
	p.crossUpdate.RLock()
	defer p.crossUpdate.RUnlock()

	p.logger.Debugf("Watcher Pod delete: %+v", obj)
	p.emit(obj.(*kubernetes.Pod), "stop")
}

// GenerateHints creates hints needed for hints builder.
func (p *pod) GenerateHints(event bus.Event) bus.Event {
	// Try to build a config with enabled builders. Send a provider agnostic payload.
	// Builders are Beat specific.
	e := bus.Event{}
	var kubeMeta, container mapstr.M

	annotations := make(mapstr.M, 0)
	rawMeta, ok := event["kubernetes"]
	if ok {
		kubeMeta = rawMeta.(mapstr.M)
		// The builder base config can configure any of the field values of kubernetes if need be.
		e["kubernetes"] = kubeMeta
		if rawAnn, ok := kubeMeta["annotations"]; ok {
			anns, _ := rawAnn.(mapstr.M)
			if len(anns) != 0 {
				annotations = anns.Clone()
			}
		}

		// Look at all the namespace level default annotations and do a merge with priority going to the pod annotations.
		if rawNsAnn, ok := kubeMeta["namespace_annotations"]; ok {
			namespaceAnnotations, _ := rawNsAnn.(mapstr.M)
			if len(namespaceAnnotations) != 0 {
				annotations.DeepUpdateNoOverwrite(namespaceAnnotations)
			}
		}
	}
	if host, ok := event["host"]; ok {
		e["host"] = host
	}
	if port, ok := event["port"]; ok {
		e["port"] = port
	}
	if ports, ok := event["ports"]; ok {
		e["ports"] = ports
	}

	if rawCont, ok := kubeMeta["container"]; ok {
		container = rawCont.(mapstr.M)
		// This would end up adding a runtime entry into the event. This would make sure
		// that there is not an attempt to spin up a docker input for a rkt container and when a
		// rkt input exists it would be natively supported.
		e["container"] = container
	}

	cname := utils.GetContainerName(container)

	// Generate hints based on the cumulative of both namespace and pod annotations.
	hints, incorrecthints := utils.GenerateHints(annotations, cname, p.config.Prefix, AllSupportedHints)
	//We check whether the provided annotation follows the supported format and vocabulary. The check happens for annotations that have prefix co.elastic
	if len(incorrecthints) > 0 {
		for _, value := range incorrecthints {
			p.logger.Warnf("provided hint: %s/%s is not in the supported list", p.config.Prefix, value)
		}
	}
	p.logger.Debugf("Generated hints %+v", hints)

	if len(hints) != 0 {
		e["hints"] = hints
	}
	p.logger.Debugf("Generated builder event %+v", e)

	return e
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

	if p.replicasetWatcher != nil {
		err := p.replicasetWatcher.Start()
		if err != nil {
			return err
		}
	}

	if p.jobWatcher != nil {
		err := p.jobWatcher.Start()
		if err != nil {
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

	if p.replicasetWatcher != nil {
		p.replicasetWatcher.Stop()
	}

	if p.jobWatcher != nil {
		p.jobWatcher.Stop()
	}
}

// emit emits the events for the given pod according to its state and
// the given flag.
// It emits a pod event if the pod has at least a running container,
// and a container event for each one of the ports defined in each
// container.
// If a container doesn't have any defined port, it emits a single
// container event with "port" set to 0.
// "start" events are only generated for containers that have an id.
// "stop" events are always generated to ensure that configurations are
// deleted.
// If the pod is terminated, "stop" events are delayed during the grace
// period defined in `CleanupTimeout`.
// Network information is only included in events for running containers
// and for pods with at least one running container.
func (p *pod) emit(pod *kubernetes.Pod, flag string) {
	annotations := kubernetes.PodAnnotations(pod)
	labels := kubernetes.PodLabels(pod)
	namespaceAnnotations := kubernetes.PodNamespaceAnnotations(pod, p.namespaceWatcher)

	eventList := make([][]bus.Event, 0)
	portsMap := mapstr.M{}
	containers := kubernetes.GetContainersInPod(pod)
	anyContainerRunning := false
	for _, c := range containers {
		if c.Status.State.Running != nil {
			anyContainerRunning = true
		}

		events, ports := p.containerPodEvents(flag, pod, c, annotations, namespaceAnnotations, labels)
		if len(events) != 0 {
			eventList = append(eventList, events)
		}
		if len(ports) > 0 {
			portsMap.DeepUpdate(ports)
		}
	}
	if len(eventList) != 0 {
		event := p.podEvent(flag, pod, portsMap, anyContainerRunning, annotations, namespaceAnnotations, labels)
		// Ensure that the pod level event is published first to avoid
		// pod metadata overriding a valid container metadata.
		eventList = append([][]bus.Event{{event}}, eventList...)
	}

	delay := (flag == "stop" && kubernetes.PodTerminated(pod, containers))
	p.publishAll(eventList, delay)
}

// containerPodEvents creates the events for a container in a pod
// One event is created for each configured port. If there is no
// configured port, a single event is created, with the port set to 0.
// Host and port information is only included if the container is
// running.
// If the container ID is unknown, only "stop" events are generated.
// It also returns a map with the named ports.
func (p *pod) containerPodEvents(flag string, pod *kubernetes.Pod, c *kubernetes.ContainerInPod, annotations, namespaceAnnotations, labels mapstr.M) ([]bus.Event, mapstr.M) {
	if c.ID == "" && flag != "stop" {
		return nil, nil
	}

	// This must be an id that doesn't depend on the state of the container
	// so it works also on `stop` if containers have been already deleted.
	eventID := fmt.Sprintf("%s.%s", pod.GetObjectMeta().GetUID(), c.Spec.Name)

	meta := p.metagen.Generate(pod, metadata.WithFields("container.name", c.Spec.Name))

	cmeta := mapstr.M{
		"id":      c.ID,
		"runtime": c.Runtime,
		"image": mapstr.M{
			"name": c.Spec.Image,
		},
	}

	// Information that can be used in discovering a workload
	kubemetaMap, _ := meta.GetValue("kubernetes")
	kubemeta, _ := kubemetaMap.(mapstr.M)
	kubemeta = kubemeta.Clone()
	kubemeta["annotations"] = annotations
	kubemeta["labels"] = labels
	kubemeta["container"] = mapstr.M{
		"id":      c.ID,
		"name":    c.Spec.Name,
		"image":   c.Spec.Image,
		"runtime": c.Runtime,
	}
	if len(namespaceAnnotations) != 0 {
		kubemeta["namespace_annotations"] = namespaceAnnotations
	}

	ports := c.Spec.Ports
	if len(ports) == 0 {
		// Ensure that at least one event is generated for this container.
		// Set port to zero to signify that the event is from a container
		// and not from a pod.
		ports = []kubernetes.ContainerPort{{ContainerPort: 0}}
	}

	events := []bus.Event{}
	portsMap := mapstr.M{}

	ShouldPut(meta, "container", cmeta, p.logger)

	for _, port := range ports {
		event := bus.Event{
			"provider":   p.uuid,
			"id":         eventID,
			flag:         true,
			"kubernetes": kubemeta,
			// Actual metadata that will enrich the event.
			"meta": meta,
		}
		// Include network information only if the container is running,
		// so templates that need network don't generate a config.
		if c.Status.State.Running != nil {
			if port.Name != "" && port.ContainerPort != 0 {
				portsMap[port.Name] = port.ContainerPort
			}
			event["host"] = pod.Status.PodIP
			event["port"] = port.ContainerPort
		}

		events = append(events, event)
	}

	return events, portsMap
}

// podEvent creates an event for a pod.
// It only includes network information if `includeNetwork` is true.
func (p *pod) podEvent(flag string, pod *kubernetes.Pod, ports mapstr.M, includeNetwork bool, annotations, namespaceAnnotations, labels mapstr.M) bus.Event {
	meta := p.metagen.Generate(pod)

	// Information that can be used in discovering a workload
	kubemetaMap, _ := meta.GetValue("kubernetes")
	kubemeta, _ := kubemetaMap.(mapstr.M)
	kubemeta = kubemeta.Clone()
	kubemeta["annotations"] = annotations
	kubemeta["labels"] = labels
	if len(namespaceAnnotations) != 0 {
		kubemeta["namespace_annotations"] = namespaceAnnotations
	}

	// Don't set a port on the event
	event := bus.Event{
		"provider":   p.uuid,
		"id":         fmt.Sprint(pod.GetObjectMeta().GetUID()),
		flag:         true,
		"kubernetes": kubemeta,
		"meta":       meta,
	}

	// Include network information only if the pod has an IP and there is any
	// running container that could handle requests.
	if pod.Status.PodIP != "" && includeNetwork {
		event["host"] = pod.Status.PodIP
		if len(ports) > 0 {
			event["ports"] = ports
		}
	}

	return event
}

// publishAll publishes all events in the event list in the same order. If delay is true
// publishAll schedules the publication of the events after the configured `CleanupPeriod`
// and returns inmediatelly.
// Order of published events matters, so this function will always publish a given eventList
// in the same goroutine.
func (p *pod) publishAll(eventList [][]bus.Event, delay bool) {
	if delay && p.config.CleanupTimeout > 0 {
		p.logger.Debug("Publish will wait for the cleanup timeout")
		time.AfterFunc(p.config.CleanupTimeout, func() {
			p.publishAll(eventList, false)
		})
		return
	}

	for _, events := range eventList {
		p.publishFunc(events)
	}
}
