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

	"github.com/elastic/beats/libbeat/autodiscover/builder"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/common/kubernetes"
	"github.com/elastic/beats/libbeat/common/safemapstr"
	"github.com/elastic/beats/libbeat/logp"
)

type pod struct {
	uuid    uuid.UUID
	config  *Config
	metagen kubernetes.MetaGenerator
	logger  *logp.Logger
	publish func(bus.Event)
	watcher kubernetes.Watcher
}

// NewPodEventer creates an eventer that can discover and process pod objects
func NewPodEventer(uuid uuid.UUID, cfg *common.Config, client k8s.Interface, publish func(event bus.Event)) (Eventer, error) {
	metagen, err := kubernetes.NewMetaGenerator(cfg)
	if err != nil {
		return nil, err
	}

	logger := logp.NewLogger("autodiscover.pod")

	config := defaultConfig()
	err = cfg.Unpack(&config)
	if err != nil {
		return nil, err
	}

	// Ensure that node is set correctly whenever the scope is set to "node". Make sure that node is empty
	// when cluster scope is enforced.
	if config.Scope == "node" {
		config.Node = kubernetes.DiscoverKubernetesNode(config.Node, kubernetes.IsInCluster(config.KubeConfig), client)
	} else {
		config.Node = ""
	}

	logger.Debugf("Initializing a new Kubernetes watcher using node: %v", config.Node)

	watcher, err := kubernetes.NewWatcher(client, &kubernetes.Pod{}, kubernetes.WatchOptions{
		SyncTimeout: config.SyncPeriod,
		Node:        config.Node,
		Namespace:   config.Namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("couldn't create watcher for %T due to error %+v", &kubernetes.Pod{}, err)
	}

	p := &pod{
		config:  config,
		uuid:    uuid,
		publish: publish,
		metagen: metagen,
		logger:  logger,
		watcher: watcher,
	}

	watcher.AddEventHandler(p)
	return p, nil
}

// OnAdd ensures processing of service objects that are newly added
func (p *pod) OnAdd(obj interface{}) {
	p.logger.Debugf("Watcher Node add: %+v", obj)
	p.emit(obj.(*kubernetes.Pod), "start")
}

// OnUpdate emits events for a given pod depending on the state of the pod,
// if it is terminating, a stop event is scheduled, if not, a stop and a start
// events are sent sequentially to recreate the resources assotiated to the pod.
func (p *pod) OnUpdate(obj interface{}) {
	pod := obj.(*kubernetes.Pod)
	if pod.GetObjectMeta().GetDeletionTimestamp() != nil {
		p.logger.Debugf("Watcher Node update (terminating): %+v", obj)
		// Node is terminating, don't reload its configuration and ignore the event
		// if some pod is still running, we will receive more events when containers
		// terminate.
		for _, container := range pod.Status.ContainerStatuses {
			if container.State.Running != nil {
				return
			}
		}
		time.AfterFunc(p.config.CleanupTimeout, func() { p.emit(pod, "stop") })
	} else {
		p.logger.Debugf("Watcher Node update: %+v", obj)
		p.emit(pod, "stop")
		p.emit(pod, "start")
	}
}

// GenerateHints creates hints needed for hints builder
func (p *pod) OnDelete(obj interface{}) {
	p.logger.Debugf("Watcher Node delete: %+v", obj)
	time.AfterFunc(p.config.CleanupTimeout, func() { p.emit(obj.(*kubernetes.Pod), "stop") })
}

func (p *pod) GenerateHints(event bus.Event) bus.Event {
	// Try to build a config with enabled builders. Send a provider agnostic payload.
	// Builders are Beat specific.
	e := bus.Event{}
	var annotations common.MapStr
	var kubeMeta, container common.MapStr
	rawMeta, ok := event["kubernetes"]
	if ok {
		kubeMeta = rawMeta.(common.MapStr)
		// The builder base config can configure any of the field values of kubernetes if need be.
		e["kubernetes"] = kubeMeta
		if rawAnn, ok := kubeMeta["annotations"]; ok {
			annotations = rawAnn.(common.MapStr)
		}
	}
	if host, ok := event["host"]; ok {
		e["host"] = host
	}
	if port, ok := event["port"]; ok {
		e["port"] = port
	}

	if rawCont, ok := kubeMeta["container"]; ok {
		container = rawCont.(common.MapStr)
		// This would end up adding a runtime entry into the event. This would make sure
		// that there is not an attempt to spin up a docker input for a rkt container and when a
		// rkt input exists it would be natively supported.
		e["container"] = container
	}

	cname := builder.GetContainerName(container)
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
	return p.watcher.Start()
}

// Stop stops the eventer
func (p *pod) Stop() {
	p.watcher.Stop()
}

func (p *pod) emit(pod *kubernetes.Pod, flag string) {
	// Emit events for all containers
	p.emitEvents(pod, flag, pod.Spec.Containers, pod.Status.ContainerStatuses)

	// Emit events for all initContainers
	p.emitEvents(pod, flag, pod.Spec.InitContainers, pod.Status.InitContainerStatuses)
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

		cmeta := common.MapStr{
			"id":      cid,
			"name":    c.Name,
			"image":   c.Image,
			"runtime": runtimes[c.Name],
		}
		meta := p.metagen.ContainerMetadata(pod, c.Name, c.Image)

		// Information that can be used in discovering a workload
		kubemeta := meta.Clone()
		kubemeta["container"] = cmeta

		// Pass annotations to all events so that it can be used in templating and by annotation builders.
		annotations := common.MapStr{}
		for k, v := range pod.GetObjectMeta().GetAnnotations() {
			safemapstr.Put(annotations, k, v)
		}
		kubemeta["annotations"] = annotations

		// Without this check there would be overlapping configurations with and without ports.
		if len(c.Ports) == 0 {
			event := bus.Event{
				"provider":   p.uuid,
				"id":         eventID,
				flag:         true,
				"host":       host,
				"kubernetes": kubemeta,
				"meta": common.MapStr{
					"kubernetes": meta,
				},
			}
			p.publish(event)
		}

		for _, port := range c.Ports {
			event := bus.Event{
				"provider":   p.uuid,
				"id":         eventID,
				flag:         true,
				"host":       host,
				"port":       port.ContainerPort,
				"kubernetes": kubemeta,
				"meta": common.MapStr{
					"kubernetes": meta,
				},
			}
			p.publish(event)
		}
	}
}
