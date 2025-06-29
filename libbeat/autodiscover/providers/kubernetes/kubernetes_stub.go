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

//go:build aix || solaris

package kubernetes

import (
	"fmt"

	"github.com/gofrs/uuid"

	"github.com/elastic/beats/v7/libbeat/autodiscover"
	"github.com/elastic/elastic-agent-autodiscover/bus"
	"github.com/elastic/elastic-agent-autodiscover/kubernetes"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/keystore"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func init() {
	err := autodiscover.Registry.AddProvider("kubernetes", AutodiscoverBuilder)
	if err != nil {
		logp.Error(fmt.Errorf("could not add `hints` builder"))
	}
}

// Eventer allows defining ways in which kubernetes resource events are observed and processed
type Eventer interface {
	kubernetes.ResourceEventHandler
	GenerateHints(event bus.Event) bus.Event
	Start() error
	Stop()
}

// EventManager allows defining ways in which kubernetes resource events are observed and processed
type EventManager interface {
	GenerateHints(event bus.Event) bus.Event
	Start()
	Stop()
}

// Provider implements autodiscover provider for docker containers
type Provider struct {
	logger *logp.Logger
}

// AutodiscoverBuilder builds and returns an autodiscover provider
func AutodiscoverBuilder(
	beatName string,
	bus bus.Bus,
	uuid uuid.UUID,
	c *config.C,
	keystore keystore.Keystore,
) (autodiscover.Provider, error) {
	logger := logp.NewLogger("autodiscover")

	p := &Provider{
		logger: logger,
	}

	return p, nil
}

// Start for Runner interface.
func (p *Provider) Start() {
}

// Stop signals the stop channel to force the watch loop routine to stop.
func (p *Provider) Stop() {
}

// String returns a description of kubernetes autodiscover provider.
func (p *Provider) String() string {
	return "kubernetes"
}

func (p *Provider) publish(events []bus.Event) {
	if len(events) == 0 {
		return
	}

}

func ShouldPut(event mapstr.M, field string, value interface{}, logger *logp.Logger) {
	_, err := event.Put(field, value)
	if err != nil {
		logger.Debugf("Failed to put field '%s' with value '%s': %s", field, value, err)
	}
}

func ShouldDelete(event mapstr.M, field string, logger *logp.Logger) {
	err := event.Delete(field)
	if err != nil {
		logger.Debugf("Failed to delete field '%s': %s", field, err)
	}
}
