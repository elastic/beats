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

package kubernetes

import (
	"fmt"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	k8s "k8s.io/client-go/kubernetes"

	"github.com/elastic/beats/v7/libbeat/autodiscover/builder"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/bus"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes/metadata"
	"github.com/elastic/beats/v7/libbeat/common/safemapstr"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type pod struct {
	uuid             uuid.UUID
	config           *Config
	metagen          metadata.MetaGen
	logger           *logp.Logger
	publishFunc      func([]bus.Event)
	watcher          kubernetes.Watcher
	nodeWatcher      kubernetes.Watcher
	namespaceWatcher kubernetes.Watcher

	// Mutex used by configuration updates not triggered by the main watcher,
	// to avoid race conditions between cross updates and deletions.
	// Other updaters must use a write lock.
	crossUpdate sync.RWMutex
}

// NewPodEventer creates an eventer that can discover and process pod objects
func NewPodEventer(uuid uuid.UUID, cfg *common.Config, client k8s.Interface, publish func(event []bus.Event)) (Eventer, error) {
	logger := logp.NewLogger("autodiscover.pod")

	config := defaultConfig()
	err := cfg.Unpack(&config)
	if err != nil {
		return nil, err
	}

	// Ensure that node is set correctly whenever the scope is set to "node". Make sure that node is empty
	// when cluster scope is enforced.
	if config.Scope == "node" {
		config.Node = kubernetes.DiscoverKubernetesNode(logger, config.Node, kubernetes.IsInCluster(config.KubeConfig), client)
	} else {
		config.Node = ""
	}

	logger.Debugf("Initializing a new Kubernetes watcher using node: %v", config.Node)

	watcher, err := kubernetes.NewWatcher(client, &kubernetes.Pod{}, kubernetes.WatchOptions{
		SyncTimeout:  config.SyncPeriod,
		Node:         config.Node,
		Namespace:    config.Namespace,
		HonorReSyncs: true,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("couldn't create watcher for %T due to error %+v", &kubernetes.Pod{}, err)
	}

	options := kubernetes.WatchOptions{
		SyncTimeout: config.SyncPeriod,
		Node:        config.Node,
	}
	if config.Namespace != "" {
		options.Namespace = config.Namespace
	}
	metaConf := config.AddResourceMetadata
	if metaConf == nil {
		metaConf = metadata.GetDefaultResourceMetadataConfig()
	}
	nodeWatcher, err := kubernetes.NewWatcher(client, &kubernetes.Node{}, options, nil)
	if err != nil {
		logger.Errorf("couldn't create watcher for %T due to error %+v", &kubernetes.Node{}, err)
	}
	namespaceWatcher, err := kubernetes.NewWatcher(client, &kubernetes.Namespace{}, kubernetes.WatchOptions{
		SyncTimeout: config.SyncPeriod,
	}, nil)
	if err != nil {
		logger.Errorf("couldn't create watcher for %T due to error %+v", &kubernetes.Namespace{}, err)
	}
	metaGen := metadata.GetPodMetaGen(cfg, watcher, nodeWatcher, namespaceWatcher, metaConf)

	p := &pod{
		config:           config,
		uuid:             uuid,
		publishFunc:      publish,
		metagen:          metaGen,
		logger:           logger,
		watcher:          watcher,
		nodeWatcher:      nodeWatcher,
		namespaceWatcher: namespaceWatcher,
	}

	watcher.AddEventHandler(p)

	if namespaceWatcher != nil && (config.Hints.Enabled() || metaConf.Namespace.Enabled()) {
		updater := newNamespacePodUpdater(p.unlockedUpdate, watcher.Store(), &p.crossUpdate)
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
	var kubeMeta, container common.MapStr

	annotations := make(common.MapStr, 0)
	rawMeta, ok := event["kubernetes"]
	if ok {
		kubeMeta = rawMeta.(common.MapStr)
		// The builder base config can configure any of the field values of kubernetes if need be.
		e["kubernetes"] = kubeMeta
		if rawAnn, ok := kubeMeta["annotations"]; ok {
			anns, _ := rawAnn.(common.MapStr)
			if len(anns) != 0 {
				annotations = anns.Clone()
			}
		}

		// Look at all the namespace level default annotations and do a merge with priority going to the pod annotations.
		if rawNsAnn, ok := kubeMeta["namespace_annotations"]; ok {
			namespaceAnnotations, _ := rawNsAnn.(common.MapStr)
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
		container = rawCont.(common.MapStr)
		// This would end up adding a runtime entry into the event. This would make sure
		// that there is not an attempt to spin up a docker input for a rkt container and when a
		// rkt input exists it would be natively supported.
		e["container"] = container
	}

	cname := builder.GetContainerName(container)

	// Generate hints based on the cumulative of both namespace and pod annotations.
	hints := builder.GenerateHints(annotations, cname, p.config.Prefix)
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

type containerInPod struct {
	id      string
	runtime string
	spec    kubernetes.Container
	status  kubernetes.PodContainerStatus
}

// getContainersInPod returns all the containers defined in a pod and their statuses.
// It includes init and ephemeral containers.
func getContainersInPod(pod *kubernetes.Pod) []*containerInPod {
	var containers []*containerInPod
	for _, c := range pod.Spec.Containers {
		containers = append(containers, &containerInPod{spec: c})
	}
	for _, c := range pod.Spec.InitContainers {
		containers = append(containers, &containerInPod{spec: c})
	}
	for _, c := range pod.Spec.EphemeralContainers {
		c := kubernetes.Container(c.EphemeralContainerCommon)
		containers = append(containers, &containerInPod{spec: c})
	}

	statuses := make(map[string]*kubernetes.PodContainerStatus)
	mapStatuses := func(s []kubernetes.PodContainerStatus) {
		for i := range s {
			statuses[s[i].Name] = &s[i]
		}
	}
	mapStatuses(pod.Status.ContainerStatuses)
	mapStatuses(pod.Status.InitContainerStatuses)
	mapStatuses(pod.Status.EphemeralContainerStatuses)
	for _, c := range containers {
		if s, ok := statuses[c.spec.Name]; ok {
			c.id, c.runtime = kubernetes.ContainerIDWithRuntime(*s)
			c.status = *s
		}
	}

	return containers
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
	annotations := podAnnotations(pod)
	namespaceAnnotations := podNamespaceAnnotations(pod, p.namespaceWatcher)

	eventList := make([][]bus.Event, 0)
	portsMap := common.MapStr{}
	containers := getContainersInPod(pod)
	anyContainerRunning := false
	for _, c := range containers {
		if c.status.State.Running != nil {
			anyContainerRunning = true
		}

		events, ports := p.containerPodEvents(flag, pod, c, annotations, namespaceAnnotations)
		if len(events) != 0 {
			eventList = append(eventList, events)
		}
		if len(ports) > 0 {
			portsMap.DeepUpdate(ports)
		}
	}
	if len(eventList) != 0 {
		event := p.podEvent(flag, pod, portsMap, anyContainerRunning, annotations, namespaceAnnotations)
		// Ensure that the pod level event is published first to avoid
		// pod metadata overriding a valid container metadata.
		eventList = append([][]bus.Event{{event}}, eventList...)
	}

	delay := (flag == "stop" && podTerminated(pod, containers))
	p.publishAll(eventList, delay)
}

// containerPodEvents creates the events for a container in a pod
// One event is created for each configured port. If there is no
// configured port, a single event is created, with the port set to 0.
// Host and port information is only included if the container is
// running.
// If the container ID is unkown, only "stop" events are generated.
// It also returns a map with the named ports.
func (p *pod) containerPodEvents(flag string, pod *kubernetes.Pod, c *containerInPod, annotations, namespaceAnnotations common.MapStr) ([]bus.Event, common.MapStr) {
	if c.id == "" && flag != "stop" {
		return nil, nil
	}

	// This must be an id that doesn't depend on the state of the container
	// so it works also on `stop` if containers have been already deleted.
	eventID := fmt.Sprintf("%s.%s", pod.GetObjectMeta().GetUID(), c.spec.Name)

	meta := p.metagen.Generate(pod, metadata.WithFields("container.name", c.spec.Name))

	cmeta := common.MapStr{
		"id":      c.id,
		"runtime": c.runtime,
		"image": common.MapStr{
			"name": c.spec.Image,
		},
	}

	// Information that can be used in discovering a workload
	kubemeta := meta.Clone()
	kubemeta["annotations"] = annotations
	kubemeta["container"] = common.MapStr{
		"id":      c.id,
		"name":    c.spec.Name,
		"image":   c.spec.Image,
		"runtime": c.runtime,
	}
	if len(namespaceAnnotations) != 0 {
		kubemeta["namespace_annotations"] = namespaceAnnotations
	}

	ports := c.spec.Ports
	if len(ports) == 0 {
		// Ensure that at least one event is generated for this container.
		// Set port to zero to signify that the event is from a container
		// and not from a pod.
		ports = []kubernetes.ContainerPort{{ContainerPort: 0}}
	}

	var events []bus.Event
	portsMap := common.MapStr{}
	for _, port := range ports {
		event := bus.Event{
			"provider":   p.uuid,
			"id":         eventID,
			flag:         true,
			"kubernetes": kubemeta,
			// Actual metadata that will enrich the event.
			"meta": common.MapStr{
				"kubernetes": meta,
				"container":  cmeta,
			},
		}
		// Include network information only if the container is running,
		// so templates that need network don't generate a config.
		if c.status.State.Running != nil {
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
func (p *pod) podEvent(flag string, pod *kubernetes.Pod, ports common.MapStr, includeNetwork bool, annotations, namespaceAnnotations common.MapStr) bus.Event {
	meta := p.metagen.Generate(pod)

	// Information that can be used in discovering a workload
	kubemeta := meta.Clone()
	kubemeta["annotations"] = annotations
	if len(namespaceAnnotations) != 0 {
		kubemeta["namespace_annotations"] = namespaceAnnotations
	}

	// Don't set a port on the event
	event := bus.Event{
		"provider":   p.uuid,
		"id":         fmt.Sprint(pod.GetObjectMeta().GetUID()),
		flag:         true,
		"kubernetes": kubemeta,
		"meta": common.MapStr{
			"kubernetes": meta,
		},
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

// podAnnotations returns the annotations in a pod
func podAnnotations(pod *kubernetes.Pod) common.MapStr {
	annotations := common.MapStr{}
	for k, v := range pod.GetObjectMeta().GetAnnotations() {
		safemapstr.Put(annotations, k, v)
	}
	return annotations
}

// podNamespaceAnnotations returns the annotations of the namespace of the pod
func podNamespaceAnnotations(pod *kubernetes.Pod, watcher kubernetes.Watcher) common.MapStr {
	if watcher == nil {
		return nil
	}

	rawNs, ok, err := watcher.Store().GetByKey(pod.Namespace)
	if !ok || err != nil {
		return nil
	}

	namespace, ok := rawNs.(*kubernetes.Namespace)
	if !ok {
		return nil
	}

	annotations := common.MapStr{}
	for k, v := range namespace.GetAnnotations() {
		safemapstr.Put(annotations, k, v)
	}
	return annotations
}

// podTerminating returns true if a pod is marked for deletion or is in a phase beyond running.
func podTerminating(pod *kubernetes.Pod) bool {
	if pod.GetObjectMeta().GetDeletionTimestamp() != nil {
		return true
	}

	switch pod.Status.Phase {
	case kubernetes.PodRunning, kubernetes.PodPending:
	default:
		return true
	}

	return false
}

// podTerminated returns true if a pod is terminated, this method considers a
// pod as terminated if none of its containers are running (or going to be running).
func podTerminated(pod *kubernetes.Pod, containers []*containerInPod) bool {
	// Pod is not marked for termination, so it is not terminated.
	if !podTerminating(pod) {
		return false
	}

	// If any container is running, the pod is not terminated yet.
	for _, container := range containers {
		if container.status.State.Running != nil {
			return false
		}
	}

	return true
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

// podUpdaterHandlerFunc is a function that handles pod updater notifications.
type podUpdaterHandlerFunc func(interface{})

// podUpdaterStore is the interface that an object needs to implement to be
// used as a pod updater store.
type podUpdaterStore interface {
	List() []interface{}
}

// namespacePodUpdater notifies updates on pods when their namespaces are updated.
type namespacePodUpdater struct {
	handler podUpdaterHandlerFunc
	store   podUpdaterStore
	locker  sync.Locker
}

// newNamespacePodUpdater creates a namespacePodUpdater
func newNamespacePodUpdater(handler podUpdaterHandlerFunc, store podUpdaterStore, locker sync.Locker) *namespacePodUpdater {
	return &namespacePodUpdater{
		handler: handler,
		store:   store,
		locker:  locker,
	}
}

// OnUpdate handles update events on namespaces.
func (n *namespacePodUpdater) OnUpdate(obj interface{}) {
	ns, ok := obj.(*kubernetes.Namespace)
	if !ok {
		return
	}

	// n.store.List() returns a snapshot at this point. If a delete is received
	// from the main watcher, this loop may generate an update event after the
	// delete is processed, leaving configurations that would never be deleted.
	// Also this loop can miss updates, what could leave outdated configurations.
	// Avoid these issues by locking the processing of events from the main watcher.
	if n.locker != nil {
		n.locker.Lock()
		defer n.locker.Unlock()
	}
	for _, pod := range n.store.List() {
		pod, ok := pod.(*kubernetes.Pod)
		if ok && pod.Namespace == ns.Name {
			n.handler(pod)
		}
	}
}

// OnAdd handles add events on namespaces. Nothing to do, if pods are added to this
// namespace they will generate their own add events.
func (*namespacePodUpdater) OnAdd(interface{}) {}

// OnDelete handles delete events on namespaces. Nothing to do, if pods are deleted from this
// namespace they will generate their own delete events.
func (*namespacePodUpdater) OnDelete(interface{}) {}
