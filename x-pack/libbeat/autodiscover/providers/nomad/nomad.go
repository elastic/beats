// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package nomad

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"github.com/elastic/beats/v7/libbeat/autodiscover"
	"github.com/elastic/beats/v7/libbeat/autodiscover/builder"
	"github.com/elastic/beats/v7/libbeat/autodiscover/template"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/bus"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/nomad"
)

func init() {
	autodiscover.Registry.AddProvider("nomad", AutodiscoverBuilder)
}

// Provider implements autodiscover provider for docker containers
type Provider struct {
	config    *Config
	bus       bus.Bus
	uuid      uuid.UUID
	metagen   nomad.MetaGenerator
	templates template.Mapper
	builders  autodiscover.Builders
	appenders autodiscover.Appenders
	watcher   nomad.Watcher
}

// AutodiscoverBuilder builds and returns an autodiscover provider
func AutodiscoverBuilder(bus bus.Bus, uuid uuid.UUID, c *common.Config) (autodiscover.Provider, error) {
	cfgwarn.Beta("The nomad autodiscover is beta")
	config := defaultConfig()

	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}

	client, err := nomad.GetNomadClient()
	if err != nil {
		logp.Err("nomad: Couldn't create client")
		return nil, err
	}

	mapper, err := template.NewConfigMapper(config.Templates)
	if err != nil {
		return nil, err
	}

	builders, err := autodiscover.NewBuilders(config.Builders, config.Hints)
	if err != nil {
		return nil, err
	}

	appenders, err := autodiscover.NewAppenders(config.Appenders)
	if err != nil {
		return nil, err
	}

	metagen, err := nomad.NewMetaGenerator(c, client)
	if err != nil {
		return nil, err
	}

	options := nomad.WatchOptions{
		SyncTimeout:     config.WaitTime,
		RefreshInterval: config.SyncPeriod,
		Node:            config.Host,
	}

	watcher, err := nomad.NewWatcher(client, options)
	if err != nil {
		logp.Err("ERROR creating Watcher %v", err.Error())
	}

	p := &Provider{
		config:    config,
		bus:       bus,
		uuid:      uuid,
		templates: mapper,
		metagen:   metagen,
		builders:  builders,
		appenders: appenders,
		watcher:   watcher,
	}

	watcher.AddEventHandler(nomad.ResourceEventHandlerFuncs{
		AddFunc: func(obj nomad.Resource) {
			logp.Debug("nomad", "Watcher Allocation add: %+v", obj.ID)
			p.emit(&obj, "start")
		},
		UpdateFunc: func(obj nomad.Resource) {
			logp.Debug("nomad", "Watcher Allocation update: %+v", obj.ID)
			time.AfterFunc(config.CleanupTimeout, func() { p.emit(&obj, "stop") })
			time.AfterFunc(config.CleanupTimeout, func() { p.emit(&obj, "start") })
		},
		DeleteFunc: func(obj nomad.Resource) {
			logp.Debug("nomad", "Watcher Allocation delete: %+v", obj.ID)
			time.AfterFunc(config.CleanupTimeout, func() { p.emit(&obj, "stop") })
		},
	})

	return p, nil
}

// Start for Runner interface.
func (p *Provider) Start() {
	if err := p.watcher.Start(); err != nil {
		logp.Err("Error starting nomad autodiscover provider: %s", err)
	}
}

// Stop signals the stop channel to force the watch loop routine to stop.
func (p *Provider) Stop() {
	p.watcher.Stop()
}

// String returns a description of nomad autodiscover provider.
func (p *Provider) String() string {
	return "nomad"
}

func (p *Provider) emit(obj *nomad.Resource, flag string) {
	// emit one event per allocation with the embedded tasks' metadata
	nodeName := obj.NodeName

	if len(nodeName) == 0 {
		// On older versions of Nomad the NodeName property is not set, as a workaround we can use
		// the NodeID
		host, err := p.metagen.AllocationNodeName(obj.NodeID)
		if err != nil {
			logp.Err("Error fetching node information: %s", err)
		}

		// If we cannot get a host, we assume that the allocation was stopped
		if len(host) == 0 {
			return
		}

		nodeName = host
	}

	// common metadata from the entire allocation
	allocMeta := p.metagen.ResourceMetadata(*obj)

	// job metadatata merged with the task metadata
	tasks := p.metagen.GroupMeta(obj.Job)

	// emit per-task separated events
	for _, task := range tasks {
		event := bus.Event{
			"provider": p.uuid,
			"id":       fmt.Sprintf("%s-%s", obj.ID, task["name"]),
			flag:       true,
			"host":     nodeName,
			"meta": common.MapStrUnion(allocMeta, common.MapStr{
				"task": task,
			}),
		}

		p.publish(event)
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

	logp.Debug("nomad", "Publishing event: %+v", event)
	p.bus.Publish(event)
}

func (p *Provider) generateHints(event bus.Event) bus.Event {
	// Try to build a config with enabled builders. Send a provider agnostic payload. Builders are
	// Beat specific.
	e := bus.Event{}

	var tags, container common.MapStr
	var meta, tasks common.MapStr

	rawMeta, ok := event["meta"]
	if ok {
		meta = rawMeta.(common.MapStr)
		// The builder base config can configure any of the field values of nomad if need be.
		e["meta"] = meta
		if rawAnn, ok := meta["tags"]; ok {
			tags = rawAnn.(common.MapStr)

			e["tags"] = tags
		}
	}

	if host, ok := event["host"]; ok {
		e["host"] = host
	}

	// Nomad supports different runtimes, so it will not always be _container_ info, but we could add
	// metadata about the runtime driver
	if rawCont, ok := meta["container"]; ok {
		container = rawCont.(common.MapStr)
		e["container"] = container
	}

	// for hints we look at the aggregated task's meta
	if rawTasks, ok := meta["task"]; ok {
		tasks, ok = rawTasks.(common.MapStr)
		if !ok {
			logp.Info("Could not get meta for the given task: %s", rawTasks)
			return e
		}
	}

	cname := builder.GetContainerName(container)
	hints := builder.GenerateHints(tasks, cname, p.config.Prefix)

	logp.Debug("nomad", "Generated hints %+v", hints)
	if len(hints) != 0 {
		e["hints"] = hints
	}
	logp.Debug("nomad", "Generated builder event %+v", e)

	prefix := strings.Split(p.config.Prefix, ".")[0]
	tasks.Delete(prefix)

	return e
}
