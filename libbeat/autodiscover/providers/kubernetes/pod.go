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
	"time"

	"github.com/gofrs/uuid"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

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
	publish          func(bus.Event)
	watcher          kubernetes.Watcher
	nodeWatcher      kubernetes.Watcher
	namespaceWatcher kubernetes.Watcher
	namespaceStore   cache.Store
}

// NewPodEventer creates an eventer that can discover and process pod objects
func NewPodEventer(uuid uuid.UUID, cfg *common.Config, client k8s.Interface, publish func(event bus.Event)) (Eventer, error) {
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
		SyncTimeout: config.SyncPeriod,
		Node:        config.Node,
		Namespace:   config.Namespace,
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
		return nil, fmt.Errorf("couldn't create watcher for %T due to error %+v", &kubernetes.Node{}, err)
	}
	namespaceWatcher, err := kubernetes.NewWatcher(client, &kubernetes.Namespace{}, kubernetes.WatchOptions{
		SyncTimeout: config.SyncPeriod,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("couldn't create watcher for %T due to error %+v", &kubernetes.Namespace{}, err)
	}
	metaGen := metadata.GetPodMetaGen(cfg, watcher, nodeWatcher, namespaceWatcher, metaConf)

	p := &pod{
		config:           config,
		uuid:             uuid,
		publish:          publish,
		metagen:          metaGen,
		logger:           logger,
		watcher:          watcher,
		nodeWatcher:      nodeWatcher,
		namespaceWatcher: namespaceWatcher,
	}

	watcher.AddEventHandler(p)
	return p, nil
}

// OnAdd ensures processing of pod objects that are newly added
func (p *pod) OnAdd(obj interface{}) {
	p.logger.Debugf("Watcher Pod add: %+v", obj)
	p.emit(obj.(*kubernetes.Pod), "start")
}

// OnUpdate emits events for a given pod depending on the state of the pod,
// if it is terminating, a stop event is scheduled, if not, a stop and a start
// events are sent sequentially to recreate the resources assotiated to the pod.
func (p *pod) OnUpdate(obj interface{}) {
	pod := obj.(*kubernetes.Pod)

	p.logger.Debugf("Watcher Pod update for pod: %+v, status: %+v", pod.Name, pod.Status.Phase)
	switch pod.Status.Phase {
	case kubernetes.PodSucceeded, kubernetes.PodFailed:
		// If Pod is in a phase where all containers in the have terminated emit a stop event
		p.logger.Debugf("Watcher Pod update (terminated): %+v", obj)
		time.AfterFunc(p.config.CleanupTimeout, func() { p.emit(pod, "stop") })
		return
	case kubernetes.PodPending:
		p.logger.Debugf("Watcher Pod update (pending): don't know what to do with this Pod yet, skipping for now: %+v", obj)
		return
	}

	// here handle the case when a Pod is in `Terminating` phase.
	// In this case the pod is neither `PodSucceeded` nor `PodFailed` and
	// hence requires special handling.
	if pod.GetObjectMeta().GetDeletionTimestamp() != nil {
		p.logger.Debugf("Watcher Pod update (terminating): %+v", obj)
		// Pod is terminating, don't reload its configuration and ignore the event
		// if some pod is still running, we will receive more events when containers
		// terminate.
		for _, container := range pod.Status.ContainerStatuses {
			if container.State.Running != nil {
				return
			}
		}
		time.AfterFunc(p.config.CleanupTimeout, func() { p.emit(pod, "stop") })
		return
	}

	p.logger.Debugf("Watcher Pod update: %+v", obj)
	p.emit(pod, "stop")
	p.emit(pod, "start")
}

// OnDelete stops pod objects that are deleted
func (p *pod) OnDelete(obj interface{}) {
	p.logger.Debugf("Watcher Pod delete: %+v", obj)
	time.AfterFunc(p.config.CleanupTimeout, func() { p.emit(obj.(*kubernetes.Pod), "stop") })
}

// GenerateHints creates hints needed for hints builder
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
			nsAnn, _ := rawNsAnn.(common.MapStr)
			if len(nsAnn) != 0 {
				annotations.DeepUpdateNoOverwrite(nsAnn)
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

func (p *pod) emit(pod *kubernetes.Pod, flag string) {
	containers, statuses := getContainersInPod(pod)
	p.emitEvents(pod, flag, containers, statuses)
}

// getContainersInPod returns all the containers defined in a pod and their statuses.
// It includes init and ephemeral containers.
func getContainersInPod(pod *kubernetes.Pod) ([]kubernetes.Container, []kubernetes.PodContainerStatus) {
	var containers []kubernetes.Container
	var statuses []kubernetes.PodContainerStatus

	// Emit events for all containers
	containers = append(containers, pod.Spec.Containers...)
	statuses = append(statuses, pod.Status.ContainerStatuses...)

	// Emit events for all initContainers
	containers = append(containers, pod.Spec.InitContainers...)
	statuses = append(statuses, pod.Status.InitContainerStatuses...)

	// Emit events for all ephemeralContainers
	// Ephemeral containers are alpha feature in k8s and this code may require some changes, if their
	// api change in the future.
	for _, c := range pod.Spec.EphemeralContainers {
		containers = append(containers, kubernetes.Container(c.EphemeralContainerCommon))
	}
	statuses = append(statuses, pod.Status.EphemeralContainerStatuses...)

	return containers, statuses
}

func (p *pod) emitEvents(pod *kubernetes.Pod, flag string, containers []kubernetes.Container,
	containerstatuses []kubernetes.PodContainerStatus) {
	host := pod.Status.PodIP

	// If the container doesn't exist in the runtime or its network
	// is not configured, it won't have an IP. Skip it as we cannot
	// generate configs without host, and an update will arrive when
	// the container is ready.
	// If stopping, emit the event in any case to ensure cleanup.
	if host == "" && flag != "stop" {
		return
	}

	// Collect all runtimes from status information.
	containerIDs := map[string]string{}
	runtimes := map[string]string{}
	for _, c := range containerstatuses {
		// If the container is not being stopped then add the container only if it is in running state.
		// This makes sure that we dont keep tailing init container logs after they have stopped.
		// Emit the event in case that the pod is being stopped.
		if flag == "stop" || c.State.Running != nil {
			cid, runtime := kubernetes.ContainerIDWithRuntime(c)
			containerIDs[c.Name] = cid
			runtimes[c.Name] = runtime
		}
	}

	// Pass annotations to all events so that it can be used in templating and by annotation builders.
	var (
		annotations = common.MapStr{}
		nsAnn       = common.MapStr{}
		events      = make([]bus.Event, 0)
	)
	for k, v := range pod.GetObjectMeta().GetAnnotations() {
		safemapstr.Put(annotations, k, v)
	}

	if p.namespaceWatcher != nil {
		if rawNs, ok, err := p.namespaceWatcher.Store().GetByKey(pod.Namespace); ok && err == nil {
			if namespace, ok := rawNs.(*kubernetes.Namespace); ok {
				for k, v := range namespace.GetAnnotations() {
					safemapstr.Put(nsAnn, k, v)
				}
			}
		}
	}

	podPorts := common.MapStr{}
	// Emit container and port information
	for _, c := range containers {
		// If it doesn't have an ID, container doesn't exist in
		// the runtime, emit only an event if we are stopping, so
		// we are sure of cleaning up configurations.
		cid := containerIDs[c.Name]
		if cid == "" && flag != "stop" {
			continue
		}

		// This must be an id that doesn't depend on the state of the container
		// so it works also on `stop` if containers have been already deleted.
		eventID := fmt.Sprintf("%s.%s", pod.GetObjectMeta().GetUID(), c.Name)

		meta := p.metagen.Generate(
			pod,
			metadata.WithFields("container.name", c.Name),
			metadata.WithFields("container.image", c.Image),
		)

		cmeta := common.MapStr{
			"id": cid,
			"image": common.MapStr{
				"name": c.Image,
			},
			"runtime": runtimes[c.Name],
		}

		// Information that can be used in discovering a workload
		kubemeta := meta.Clone()
		kubemeta["annotations"] = annotations
		kubemeta["container"] = common.MapStr{
			"id":      cid,
			"name":    c.Name,
			"image":   c.Image,
			"runtime": runtimes[c.Name],
		}
		if len(nsAnn) != 0 {
			kubemeta["namespace_annotations"] = nsAnn
		}

		// Without this check there would be overlapping configurations with and without ports.
		if len(c.Ports) == 0 {
			// Set a zero port on the event to signify that the event is from a container
			event := bus.Event{
				"provider":   p.uuid,
				"id":         eventID,
				flag:         true,
				"host":       host,
				"port":       0,
				"kubernetes": kubemeta,
				"meta": common.MapStr{
					"kubernetes": meta,
					"container":  cmeta,
				},
			}
			events = append(events, event)
		}

		for _, port := range c.Ports {
			podPorts[port.Name] = port.ContainerPort
			event := bus.Event{
				"provider":   p.uuid,
				"id":         eventID,
				flag:         true,
				"host":       host,
				"port":       port.ContainerPort,
				"kubernetes": kubemeta,
				"meta": common.MapStr{
					"kubernetes": meta,
					"container":  cmeta,
				},
			}
			events = append(events, event)
		}
	}

	// Publish a pod level event so that hints that have no exposed ports can get processed.
	// Log hints would just ignore this event as there is no ${data.container.id}
	// Publish the pod level hint only if at least one container level hint was generated. This ensures that there is
	// no unnecessary pod level events emitted prematurely.
	// We publish the pod level hint first so that it doesn't override a valid container level event.
	if len(events) != 0 {
		meta := p.metagen.Generate(pod)

		// Information that can be used in discovering a workload
		kubemeta := meta.Clone()
		kubemeta["annotations"] = annotations
		if len(nsAnn) != 0 {
			kubemeta["namespace_annotations"] = nsAnn
		}

		// Don't set a port on the event
		event := bus.Event{
			"provider":   p.uuid,
			"id":         fmt.Sprint(pod.GetObjectMeta().GetUID()),
			flag:         true,
			"host":       host,
			"ports":      podPorts,
			"kubernetes": kubemeta,
			"meta": common.MapStr{
				"kubernetes": meta,
			},
		}
		p.publish(event)
	}

	// Publish the container level hints finally.
	for _, event := range events {
		p.publish(event)
	}
}
