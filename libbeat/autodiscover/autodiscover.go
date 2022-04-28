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

package autodiscover

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/autodiscover/meta"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common/bus"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/keystore"
	"github.com/elastic/beats/v7/libbeat/logp"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	// If a config reload fails after a new event, a new reload will be run after this period
	retryPeriod = 10 * time.Second
)

// EventConfigurer is used to configure the creation of configuration objects
// from the autodiscover event bus.
type EventConfigurer interface {
	// EventFilter returns the bus filter to retrieve runner start/stop triggering
	// events. The bus will filter events to the ones, that contain *all* the
	// the required top-level keys.
	EventFilter() []string

	// CreateConfig creates a list of configurations from a bus.Event. The
	// received event will have all keys defined in `EventFilter`.
	CreateConfig(bus.Event) ([]*conf.C, error)
}

// Autodiscover process, it takes a beat adapter and user config and runs autodiscover process, spawning
// new modules when any configured providers does a match
type Autodiscover struct {
	bus             bus.Bus
	defaultPipeline beat.PipelineConnector
	factory         cfgfile.RunnerFactory
	configurer      EventConfigurer
	providers       []Provider
	configs         map[string]map[uint64]*reload.ConfigWithMeta
	runners         *cfgfile.RunnerList
	meta            *meta.Map
	listener        bus.Listener
	logger          *logp.Logger

	// workDone is a channel used for testing purpouses, to know when the worker has
	// done some work.
	workDone chan struct{}
}

// NewAutodiscover instantiates and returns a new Autodiscover manager
func NewAutodiscover(
	name string,
	pipeline beat.PipelineConnector,
	factory cfgfile.RunnerFactory,
	configurer EventConfigurer,
	c *Config,
	keystore keystore.Keystore,
) (*Autodiscover, error) {
	logger := logp.NewLogger("autodiscover")

	// Init Event bus
	bus := bus.New(logger, name)

	// Init providers
	var providers []Provider
	for _, providerCfg := range c.Providers {
		provider, err := Registry.BuildProvider(name, bus, providerCfg, keystore)
		if err != nil {
			return nil, errors.Wrap(err, "error in autodiscover provider settings")
		}
		logger.Debugf("Configured autodiscover provider: %s", provider)
		providers = append(providers, provider)
	}

	return &Autodiscover{
		bus:             bus,
		defaultPipeline: pipeline,
		factory:         factory,
		configurer:      configurer,
		configs:         map[string]map[uint64]*reload.ConfigWithMeta{},
		runners:         cfgfile.NewRunnerList("autodiscover", factory, pipeline),
		providers:       providers,
		meta:            meta.NewMap(),
		logger:          logger,
	}, nil
}

// Start autodiscover process
func (a *Autodiscover) Start() {
	if a == nil {
		return
	}

	a.logger.Info("Starting autodiscover manager")
	a.listener = a.bus.Subscribe(a.configurer.EventFilter()...)

	// It is important to start the worker first before starting the producer.
	// In hosts that have large number of workloads, it is easy to have an initial
	// sync of workloads to have a count that is greater than 100 (which is the size
	// of the bounded Go channel. Starting the providers before the consumer would
	// result in the channel filling up and never allowing the worker to start up.
	go a.worker()

	for _, provider := range a.providers {
		provider.Start()
	}
}

func (a *Autodiscover) worker() {
	var updated, retry bool

	for {
		select {
		case event := <-a.listener.Events():
			// This will happen on Stop:
			if event == nil {
				return
			}

			if _, ok := event["start"]; ok {
				updated = a.handleStart(event)
			}
			if _, ok := event["stop"]; ok {
				updated = a.handleStop(event)
			}

		case <-time.After(retryPeriod):
		}

		if updated || retry {
			if retry {
				a.logger.Debug("Reloading existing autodiscover configs after error")
			}

			configs := []*reload.ConfigWithMeta{}
			for _, list := range a.configs {
				for _, c := range list {
					configs = append(configs, c)
				}
			}

			err := a.runners.Reload(configs)

			// On error, make sure the next run also updates because some runners were not properly loaded
			retry = err != nil
			// reset updated status
			updated = false
		}

		// For testing purpouses.
		if a.workDone != nil {
			a.workDone <- struct{}{}
		}
	}
}

func (a *Autodiscover) handleStart(event bus.Event) bool {
	a.logger.Debugw("Got a start event.", "autodiscover.event", event)

	eventID := getID(event)
	if eventID == "" {
		a.logger.Errorf("Event didn't provide instance id: %+v, ignoring it", event)
		return false
	}

	// Ensure configs list exists for this instance
	if _, ok := a.configs[eventID]; !ok {
		a.configs[eventID] = make(map[uint64]*reload.ConfigWithMeta)
	}

	configs, err := a.configurer.CreateConfig(event)
	if err != nil {
		a.logger.Debugf("Could not generate config from event %v: %v", event, err)
		return false
	}

	if a.logger.IsDebug() {
		for _, c := range configs {
			a.logger.Debugf("Generated config: %+v", conf.DebugString(c, true))
		}
	}

	var (
		updated bool
		newCfg  = make(map[uint64]*reload.ConfigWithMeta)
	)

	meta := a.getMeta(event)
	for _, config := range configs {
		hash, err := cfgfile.HashConfig(config)
		if err != nil {
			a.logger.Debugf("Could not hash config %v: %v", conf.DebugString(config, true), err)
			continue
		}

		// Update meta no matter what
		dynFields := a.meta.Store(hash, meta)

		if _, ok := newCfg[hash]; ok {
			a.logger.Debugf("Config %v duplicated in start event", conf.DebugString(config, true))
			continue
		}

		if cfg, ok := a.configs[eventID][hash]; ok {
			a.logger.Debugf("Config %v is already running", conf.DebugString(config, true))
			newCfg[hash] = cfg
			continue
		}

		err = a.factory.CheckConfig(config)
		if err != nil {
			a.logger.Error(errors.Wrap(err, fmt.Sprintf(
				"Auto discover config check failed for config '%s', won't start runner",
				conf.DebugString(config, true))))
			continue
		}
		newCfg[hash] = &reload.ConfigWithMeta{
			Config: config,
			Meta:   &dynFields,
		}

		updated = true
	}

	// If the new add event has lesser configs than the previous stable configuration then it means that there were
	// configs that were removed in something like a resync event.
	if len(newCfg) < len(a.configs[eventID]) {
		updated = true
	}

	// By replacing the config's for eventID we make sure that all old configs that are no longer in use
	// are stopped correctly. This will ensure that a resync event is handled correctly.
	if updated {
		a.configs[eventID] = newCfg
	}

	return updated
}

func (a *Autodiscover) handleStop(event bus.Event) bool {
	var updated bool

	a.logger.Debugf("Got a stop event: %v", event)
	eventID := getID(event)
	if eventID == "" {
		a.logger.Errorf("Event didn't provide instance id: %+v, ignoring it", event)
		return false
	}

	if len(a.configs[eventID]) > 0 {
		a.logger.Debugf("Stopping %d configs", len(a.configs[eventID]))
		updated = true
	}

	delete(a.configs, eventID)

	return updated
}

func (a *Autodiscover) getMeta(event bus.Event) mapstr.M {
	m := event["meta"]
	if m == nil {
		return nil
	}

	a.logger.Debugf("Got a meta field in the event")
	meta, ok := m.(mapstr.M)
	if !ok {
		a.logger.Errorf("Got a wrong meta field for event %v", event)
		return nil
	}
	return meta
}

// getID returns the event "id" field string if present
func getID(e bus.Event) string {
	provider, ok := e["provider"]
	if !ok {
		return ""
	}

	id, ok := e["id"]
	if !ok {
		return ""
	}

	return fmt.Sprintf("%s:%s", provider, id)
}

// Stop autodiscover process
func (a *Autodiscover) Stop() {
	if a == nil {
		return
	}

	// Stop listening for events
	a.listener.Stop()

	// Stop providers
	for _, provider := range a.providers {
		provider.Stop()
	}

	// Stop runners
	a.runners.Stop()
	a.logger.Info("Stopped autodiscover manager")
}
