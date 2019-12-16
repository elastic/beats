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

package kubernetes

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/autodiscover"
	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/common/kubernetes"
	"github.com/elastic/beats/libbeat/logp"
)

func init() {
	autodiscover.Registry.AddProvider("kubernetes", AutodiscoverBuilder)
}

// Eventer allows defining ways in which kubernetes resource events are observed and processed
type Eventer interface {
	kubernetes.ResourceEventHandler
	GenerateHints(event bus.Event) bus.Event
	Start() error
	Stop()
}

// Provider implements autodiscover provider for docker containers
type Provider struct {
	config    *Config
	bus       bus.Bus
	templates template.Mapper
	builders  autodiscover.Builders
	appenders autodiscover.Appenders
	logger    *logp.Logger
	eventer   Eventer
}

// AutodiscoverBuilder builds and returns an autodiscover provider
func AutodiscoverBuilder(bus bus.Bus, uuid uuid.UUID, c *common.Config) (autodiscover.Provider, error) {
	cfgwarn.Beta("The kubernetes autodiscover is beta")
	logger := logp.NewLogger("autodiscover")

	errWrap := func(err error) error {
		return errors.Wrap(err, "error setting up kubernetes autodiscover provider")
	}

	config := defaultConfig()
	err := c.Unpack(&config)
	if err != nil {
		return nil, errWrap(err)
	}

	client, err := kubernetes.GetKubernetesClient(config.KubeConfig)
	if err != nil {
		return nil, errWrap(err)
	}

	mapper, err := template.NewConfigMapper(config.Templates)
	if err != nil {
		return nil, errWrap(err)
	}

	builders, err := autodiscover.NewBuilders(config.Builders, config.Hints)
	if err != nil {
		return nil, errWrap(err)
	}

	appenders, err := autodiscover.NewAppenders(config.Appenders)
	if err != nil {
		return nil, errWrap(err)
	}

	p := &Provider{
		config:    config,
		bus:       bus,
		templates: mapper,
		builders:  builders,
		appenders: appenders,
		logger:    logger,
	}

	switch config.Resource {
	case "pod":
		p.eventer, err = NewPodEventer(uuid, c, client, p.publish)
	case "node":
		p.eventer, err = NewNodeEventer(uuid, c, client, p.publish)
	case "service":
		p.eventer, err = NewServiceEventer(uuid, c, client, p.publish)
	default:
		return nil, fmt.Errorf("unsupported autodiscover resource %s", config.Resource)
	}

	if err != nil {
		return nil, errWrap(err)
	}

	return p, nil
}

// Start for Runner interface.
func (p *Provider) Start() {
	if err := p.eventer.Start(); err != nil {
		p.logger.Errorf("Error starting kubernetes autodiscover provider: %s", err)
	}
}

// Stop signals the stop channel to force the watch loop routine to stop.
func (p *Provider) Stop() {
	p.eventer.Stop()
}

// String returns a description of kubernetes autodiscover provider.
func (p *Provider) String() string {
	return "kubernetes"
}

func (p *Provider) publish(event bus.Event) {
	// Try to match a config
	if config := p.templates.GetConfig(event); config != nil {
		event["config"] = config
	} else {
		// If there isn't a default template then attempt to use builders
		if config := p.builders.GetConfig(p.eventer.GenerateHints(event)); config != nil {
			event["config"] = config
		}
	}

	// Call all appenders to append any extra configuration
	p.appenders.Append(event)
	p.bus.Publish(event)
}
