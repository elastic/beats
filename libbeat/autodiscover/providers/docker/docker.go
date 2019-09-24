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

// +build linux darwin windows

package docker

import (
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/autodiscover/builder"
	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/common/docker"
	"github.com/elastic/beats/libbeat/common/safemapstr"
	"github.com/elastic/beats/libbeat/logp"
)

func init() {
	autodiscover.Registry.AddProvider("docker", AutodiscoverBuilder)
}

// Provider implements autodiscover provider for docker containers
type Provider struct {
	config        *Config
	bus           bus.Bus
	uuid          uuid.UUID
	builders      autodiscover.Builders
	appenders     autodiscover.Appenders
	watcher       docker.Watcher
	templates     template.Mapper
	stop          chan interface{}
	startListener bus.Listener
	stopListener  bus.Listener
	stoppers      map[string]*time.Timer
	stopTrigger   chan *dockerContainerMetadata
	logger        *logp.Logger
}

// AutodiscoverBuilder builds and returns an autodiscover provider
func AutodiscoverBuilder(bus bus.Bus, uuid uuid.UUID, c *common.Config) (autodiscover.Provider, error) {
	cfgwarn.Beta("The docker autodiscover is beta")

	errWrap := func(err error) error {
		return errors.Wrap(err, "error setting up docker autodiscover provider")
	}

	config := defaultConfig()
	err := c.Unpack(&config)
	if err != nil {
		return nil, errWrap(err)
	}

	watcher, err := docker.NewWatcher(config.Host, config.TLS, false)
	if err != nil {
		return nil, errWrap(err)
	}

	mapper, err := template.NewConfigMapper(config.Templates)
	if err != nil {
		return nil, errWrap(err)
	}
	if len(mapper) == 0 && !config.Hints.Enabled() {
		return nil, errWrap(fmt.Errorf("no configs or hints defined for autodiscover provider"))
	}

	builders, err := autodiscover.NewBuilders(config.Builders, config.Hints)
	if err != nil {
		return nil, errWrap(err)
	}

	appenders, err := autodiscover.NewAppenders(config.Appenders)
	if err != nil {
		return nil, errWrap(err)
	}

	start := watcher.ListenStart()
	stop := watcher.ListenStop()

	if err := watcher.Start(); err != nil {
		return nil, errWrap(err)
	}

	return &Provider{
		config:        config,
		bus:           bus,
		uuid:          uuid,
		builders:      builders,
		appenders:     appenders,
		templates:     mapper,
		watcher:       watcher,
		stop:          make(chan interface{}),
		startListener: start,
		stopListener:  stop,
		stoppers:      make(map[string]*time.Timer),
		stopTrigger:   make(chan *dockerContainerMetadata),
		logger:        logp.NewLogger("docker"),
	}, nil
}

// Start the autodiscover process
func (d *Provider) Start() {
	go func() {
		for {
			select {
			case <-d.stop:
				d.startListener.Stop()
				d.stopListener.Stop()

				// Stop all timers before closing the channel
				for _, stopper := range d.stoppers {
					stopper.Stop()
				}
				close(d.stopTrigger)
				return

			case event := <-d.startListener.Events():
				d.startContainer(event)

			case event := <-d.stopListener.Events():
				d.scheduleStopContainer(event)

			case target := <-d.stopTrigger:
				d.stopContainer(target.container, target.metadata)
			}
		}
	}()
}

type dockerContainerMetadata struct {
	container *docker.Container
	metadata  *dockerMetadata
}

type dockerMetadata struct {
	// Old selectors [Deprecated]
	Docker common.MapStr

	// New ECS-based selectors
	Container common.MapStr

	// Metadata used to enrich events, like ECS-based selectors but can
	// have modifications like dedotting
	Metadata common.MapStr
}

func (d *Provider) generateMetaDocker(event bus.Event) (*docker.Container, *dockerMetadata) {
	container, ok := event["container"].(*docker.Container)
	if !ok {
		d.logger.Error(errors.New("Couldn't get a container from watcher event"))
		return nil, nil
	}

	// Don't dedot selectors, dedot only metadata used for events enrichment
	labelMap := common.MapStr{}
	metaLabelMap := common.MapStr{}
	for k, v := range container.Labels {
		safemapstr.Put(labelMap, k, v)
		if d.config.Dedot {
			label := common.DeDot(k)
			metaLabelMap.Put(label, v)
		} else {
			safemapstr.Put(metaLabelMap, k, v)
		}
	}

	meta := &dockerMetadata{
		Docker: common.MapStr{
			"container": common.MapStr{
				"id":     container.ID,
				"name":   container.Name,
				"image":  container.Image,
				"labels": labelMap,
			},
		},
		Container: common.MapStr{
			"id":   container.ID,
			"name": container.Name,
			"image": common.MapStr{
				"name": container.Image,
			},
			"labels": labelMap,
		},
		Metadata: common.MapStr{
			"container": common.MapStr{
				"id":   container.ID,
				"name": container.Name,
				"image": common.MapStr{
					"name": container.Image,
				},
			},
			"docker": common.MapStr{
				"container": common.MapStr{
					"labels": metaLabelMap,
				},
			},
		},
	}

	return container, meta
}

func (d *Provider) startContainer(event bus.Event) {
	container, meta := d.generateMetaDocker(event)
	if container == nil || meta == nil {
		return
	}

	if stopper, ok := d.stoppers[container.ID]; ok {
		d.logger.Debugf("Container %s is restarting, aborting pending stop", container.ID)
		stopper.Stop()
		delete(d.stoppers, container.ID)
		return
	}

	d.emitContainer(container, meta, "start")
}

func (d *Provider) scheduleStopContainer(event bus.Event) {
	container, meta := d.generateMetaDocker(event)
	if container == nil || meta == nil {
		return
	}

	if d.config.CleanupTimeout <= 0 {
		d.stopContainer(container, meta)
		return
	}

	stopper := time.AfterFunc(d.config.CleanupTimeout, func() {
		d.stopTrigger <- &dockerContainerMetadata{
			container: container,
			metadata:  meta,
		}
	})
	d.stoppers[container.ID] = stopper
}

func (d *Provider) stopContainer(container *docker.Container, meta *dockerMetadata) {
	if _, ok := d.stoppers[container.ID]; ok {
		delete(d.stoppers, container.ID)
	}

	d.emitContainer(container, meta, "stop")
}

func (d *Provider) emitContainer(container *docker.Container, meta *dockerMetadata, flag string) {
	var host string
	if len(container.IPAddresses) > 0 {
		host = container.IPAddresses[0]
	}

	// Without this check there would be overlapping configurations with and without ports.
	if len(container.Ports) == 0 {
		event := bus.Event{
			"provider":  d.uuid,
			"id":        container.ID,
			flag:        true,
			"host":      host,
			"docker":    meta.Docker,
			"container": meta.Container,
			"meta":      meta.Metadata,
		}

		d.publish(event)
	}

	// Emit container container and port information
	for _, port := range container.Ports {
		event := bus.Event{
			"provider":  d.uuid,
			"id":        container.ID,
			flag:        true,
			"host":      host,
			"port":      port.PrivatePort,
			"docker":    meta.Docker,
			"container": meta.Container,
			"meta":      meta.Metadata,
		}

		d.publish(event)
	}
}

func (d *Provider) publish(event bus.Event) {
	// Try to match a config
	if config := d.templates.GetConfig(event); config != nil {
		event["config"] = config
	} else {
		// If no template matches, try builders:
		if config := d.builders.GetConfig(d.generateHints(event)); config != nil {
			event["config"] = config
		}
	}

	// Call all appenders to append any extra configuration
	d.appenders.Append(event)

	d.bus.Publish(event)
}

func (d *Provider) generateHints(event bus.Event) bus.Event {
	// Try to build a config with enabled builders. Send a provider agnostic payload.
	// Builders are Beat specific.
	e := bus.Event{}
	var dockerMeta common.MapStr

	if rawDocker, err := common.MapStr(event).GetValue("docker.container"); err == nil {
		dockerMeta = rawDocker.(common.MapStr)
		e["container"] = dockerMeta
	}

	if host, ok := event["host"]; ok {
		e["host"] = host
	}
	if port, ok := event["port"]; ok {
		e["port"] = port
	}
	if labels, err := dockerMeta.GetValue("labels"); err == nil {
		hints := builder.GenerateHints(labels.(common.MapStr), "", d.config.Prefix)
		e["hints"] = hints
	}
	return e
}

// Stop the autodiscover process
func (d *Provider) Stop() {
	close(d.stop)
}

func (d *Provider) String() string {
	return "docker"
}
