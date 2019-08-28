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

	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/autodiscover/builder"
	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/common/kubernetes"
	"github.com/elastic/beats/libbeat/common/safemapstr"
	"github.com/elastic/beats/libbeat/logp"
)

func init() {
	autodiscover.Registry.AddProvider("kubernetes", AutodiscoverBuilder)
}

// Provider implements autodiscover provider for docker containers
type Provider struct {
	config    *Config
	bus       bus.Bus
	uuid      uuid.UUID
	watcher   kubernetes.Watcher
	metagen   kubernetes.MetaGenerator
	templates template.Mapper
	builders  autodiscover.Builders
	appenders autodiscover.Appenders
}

// AutodiscoverBuilder builds and returns an autodiscover provider
func AutodiscoverBuilder(bus bus.Bus, uuid uuid.UUID, c *common.Config) (autodiscover.Provider, error) {
	cfgwarn.Beta("The kubernetes autodiscover is beta")
	config := defaultConfig()
	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.GetKubernetesClient(config.InCluster, config.KubeConfig)
	if err != nil {
		return nil, err
	}

	metagen, err := kubernetes.NewMetaGenerator(c)
	if err != nil {
		return nil, err
	}

	config.Host = kubernetes.DiscoverKubernetesNode(config.Host, config.InCluster, client)

	watcher, err := kubernetes.NewWatcher(client, &kubernetes.Pod{}, kubernetes.WatchOptions{
		SyncTimeout: config.SyncPeriod,
		Node:        config.Host,
		Namespace:   config.Namespace,
	})
	if err != nil {
		logp.Err("kubernetes: Couldn't create watcher for %t", &kubernetes.Pod{})
		return nil, err
	}

	mapper, err := template.NewConfigMapper(config.Templates)
	if err != nil {
		return nil, err
	}

	builders, err := autodiscover.NewBuilders(config.Builders, config.HintsEnabled)
	if err != nil {
		return nil, err
	}

	appenders, err := autodiscover.NewAppenders(config.Appenders)
	if err != nil {
		return nil, err
	}

	p := &Provider{
		config:    config,
		bus:       bus,
		uuid:      uuid,
		templates: mapper,
		builders:  builders,
		appenders: appenders,
		metagen:   metagen,
		watcher:   watcher,
	}

	watcher.AddEventHandler(kubernetes.ResourceEventHandlerFuncs{
		AddFunc: func(obj kubernetes.Resource) {
			logp.Debug("kubernetes", "Watcher Pod add: %+v", obj)
			p.emit(obj.(*kubernetes.Pod), "start")
		},
		UpdateFunc: func(obj kubernetes.Resource) {
			logp.Debug("kubernetes", "Watcher Pod update: %+v", obj)
			p.emit(obj.(*kubernetes.Pod), "stop")
			p.emit(obj.(*kubernetes.Pod), "start")
		},
		DeleteFunc: func(obj kubernetes.Resource) {
			logp.Debug("kubernetes", "Watcher Pod delete: %+v", obj)
			time.AfterFunc(config.CleanupTimeout, func() { p.emit(obj.(*kubernetes.Pod), "stop") })
		},
	})

	return p, nil
}

// Start for Runner interface.
func (p *Provider) Start() {
	if err := p.watcher.Start(); err != nil {
		logp.Err("Error starting kubernetes autodiscover provider: %s", err)
	}
}

func (p *Provider) emit(pod *kubernetes.Pod, flag string) {
	// Emit events for all containers
	p.emitEvents(pod, flag, pod.Spec.Containers, pod.Status.ContainerStatuses)

	// Emit events for all initContainers
	p.emitEvents(pod, flag, pod.Spec.InitContainers, pod.Status.InitContainerStatuses)
}

func (p *Provider) emitEvents(pod *kubernetes.Pod, flag string, containers []*kubernetes.Container,
	containerstatuses []*kubernetes.PodContainerStatus) {
	host := pod.Status.GetPodIP()

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
		cid, runtime := kubernetes.ContainerIDWithRuntime(c)
		containerIDs[c.GetName()] = cid
		runtimes[c.GetName()] = runtime
	}

	// Emit container and port information
	for _, c := range containers {
		// If it doesn't have an ID, container doesn't exist in
		// the runtime, emit only an event if we are stopping, so
		// we are sure of cleaning up configurations.
		cid := containerIDs[c.GetName()]
		if cid == "" && flag != "stop" {
			continue
		}

		// This must be an id that doesn't depend on the state of the container
		// so it works also on `stop` if containers have been already deleted.
		eventID := fmt.Sprintf("%s.%s", pod.Metadata.GetUid(), c.GetName())

		cmeta := common.MapStr{
			"id":      cid,
			"name":    c.GetName(),
			"image":   c.GetImage(),
			"runtime": runtimes[c.GetName()],
		}
		meta := p.metagen.ContainerMetadata(pod, c.GetName(), c.GetImage())

		// Information that can be used in discovering a workload
		kubemeta := meta.Clone()
		kubemeta["container"] = cmeta

		// Pass annotations to all events so that it can be used in templating and by annotation builders.
		annotations := common.MapStr{}
		for k, v := range pod.GetMetadata().Annotations {
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
				"port":       port.GetContainerPort(),
				"kubernetes": kubemeta,
				"meta": common.MapStr{
					"kubernetes": meta,
				},
			}
			p.publish(event)
		}
	}
}

func (p *Provider) publish(event bus.Event) {
	// Try to match a config
	if config := p.templates.GetConfig(event); config != nil {
		event["config"] = config
	} else {
		// If there isn't a default template then attempt to use builders
		if config := p.builders.GetConfig(p.generateHints(event)); config != nil {
			event["config"] = config
		}
	}

	// Call all appenders to append any extra configuration
	p.appenders.Append(event)
	p.bus.Publish(event)
}

func (p *Provider) generateHints(event bus.Event) bus.Event {
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
	if len(hints) != 0 {
		e["hints"] = hints
	}

	logp.Debug("kubernetes", "Generated builder event %v", event)

	return e
}

// Stop signals the stop channel to force the watch loop routine to stop.
func (p *Provider) Stop() {
	p.watcher.Stop()
}

// String returns a description of kubernetes autodiscover provider.
func (p *Provider) String() string {
	return "kubernetes"
}
