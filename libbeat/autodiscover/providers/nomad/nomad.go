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

package nomad

import (
	"fmt"
	"os"
	"time"

	"github.com/gofrs/uuid"

	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/autodiscover/builder"
	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/common/nomad"
	"github.com/elastic/beats/libbeat/logp"
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

	metagen, err := nomad.NewMetaGenerator(c)
	if err != nil {
		return nil, err
	}

	host, err := os.Hostname()
	if err != nil || len(host) == 0 {
		return nil, fmt.Errorf("Error getting the hostname: %v", err)
	}

	options := nomad.WatchOptions{
		SyncTimeout: 1 * time.Second,
		Node:        host,
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

	// TODO
	// add an event handler to deal with the "events" receveid from the API
	watcher.AddEventHandler(nomad.ResourceEventHandlerFuncs{
		AddFunc: func(obj nomad.Resource) {
			logp.Debug("nomad", "Watcher Allocation add: %+v", obj)
			p.emit(&obj, "start")
		},
		UpdateFunc: func(obj nomad.Resource) {
			logp.Debug("nomad", "Watcher Allocation update: %+v", obj)
			p.emit(&obj, "stop")
			p.emit(&obj, "start")
		},
		DeleteFunc: func(obj nomad.Resource) {
			logp.Debug("nomad", "Watcher Allocation delete: %+v", obj)
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

// String returns a description of kubernetes autodiscover provider.
func (p *Provider) String() string {
	return "nomad"
}

func (p *Provider) emit(obj *nomad.Resource, flag string) {
	// emit one event per allocation with the embedded tasks' metadata
	objMeta := p.metagen.ResourceMetadata(*obj)

	for _, group := range obj.Job.TaskGroups {
		for range group.Tasks {
			event := bus.Event{
				"provider": p.uuid,
				"id":       obj.ID,
				flag:       true, // event type
				// "task":     task.Name,
				"meta": objMeta,
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

	var tags common.MapStr
	var meta, container common.MapStr

	rawMeta, ok := event["meta"]
	if ok {
		meta = rawMeta.(common.MapStr)
		// The builder base config can configure any of the field values of kubernetes if need be.
		e["meta"] = meta
		if rawAnn, ok := meta["tags"]; ok {
			tags = rawAnn.(common.MapStr)

			e["tags"] = tags
		}
	}

	if host, ok := event["host"]; ok {
		e["host"] = host
	}

	// We keep this in case that we decide to add information about the container.
	// Nomad supports different runtimes, so it will not always be _container_ info, but we could
	// generalize it by calling it runtime
	if rawCont, ok := meta["container"]; ok {
		container = rawCont.(common.MapStr)
		// This would end up adding a runtime entry into the event. This would make sure
		// that there is not an attempt to spin up a docker input for a rkt container and when a
		// rkt input exists it would be natively supported.
		e["container"] = container
	}

	cname := builder.GetContainerName(container)
	hints := builder.GenerateHints(meta, cname, p.config.Prefix)

	logp.Debug("nomad", "Generated hints %+v", hints)
	if len(hints) != 0 {
		e["hints"] = hints
	}
	logp.Debug("nomad", "Generated builder event %+v", e)

	return e
}
