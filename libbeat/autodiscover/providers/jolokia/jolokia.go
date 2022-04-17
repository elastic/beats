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

package jolokia

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/autodiscover"
	"github.com/menderesk/beats/v7/libbeat/autodiscover/template"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/bus"
	"github.com/menderesk/beats/v7/libbeat/keystore"
)

func init() {
	autodiscover.Registry.AddProvider("jolokia", AutodiscoverBuilder)
}

// DiscoveryProber implements discovery probes
type DiscoveryProber interface {
	Start()
	Stop()
	Events() <-chan Event
}

// Provider is the Jolokia Discovery autodiscover provider
type Provider struct {
	config    *Config
	bus       bus.Bus
	builders  autodiscover.Builders
	appenders autodiscover.Appenders
	templates template.Mapper
	discovery DiscoveryProber
}

// AutodiscoverBuilder builds a Jolokia Discovery autodiscover provider, it fails if
// there is some problem with the configuration
func AutodiscoverBuilder(
	beatName string,
	bus bus.Bus,
	uuid uuid.UUID,
	c *common.Config,
	keystore keystore.Keystore,
) (autodiscover.Provider, error) {
	errWrap := func(err error) error {
		return errors.Wrap(err, "error setting up jolokia autodiscover provider")
	}

	config := defaultConfig()
	err := c.Unpack(&config)
	if err != nil {
		return nil, errWrap(err)
	}

	discovery := &Discovery{
		ProviderUUID: uuid,
		Interfaces:   config.Interfaces,
	}

	mapper, err := template.NewConfigMapper(config.Templates, keystore, nil)
	if err != nil {
		return nil, errWrap(err)
	}
	if len(mapper.ConditionMaps) == 0 {
		return nil, errWrap(fmt.Errorf("no configs defined for autodiscover provider"))
	}

	builders, err := autodiscover.NewBuilders(config.Builders, nil, nil)
	if err != nil {
		return nil, errWrap(err)
	}

	appenders, err := autodiscover.NewAppenders(config.Appenders)
	if err != nil {
		return nil, errWrap(err)
	}

	return &Provider{
		bus:       bus,
		templates: mapper,
		builders:  builders,
		appenders: appenders,
		discovery: discovery,
	}, nil
}

// Start starts autodiscover provider
func (p *Provider) Start() {
	p.discovery.Start()
	go func() {
		for event := range p.discovery.Events() {
			p.publish(event.BusEvent())
		}
	}()
}

func (p *Provider) publish(event bus.Event) {
	if config := p.templates.GetConfig(event); config != nil {
		event["config"] = config
	} else if config := p.builders.GetConfig(event); config != nil {
		event["config"] = config
	}

	p.appenders.Append(event)
	p.bus.Publish(event)
}

// Stop stops autodiscover provider
func (p *Provider) Stop() {
	p.discovery.Stop()
}

// String returns the name of the provider
func (p *Provider) String() string {
	return "jolokia"
}
