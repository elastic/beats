// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package composable

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
)

// Vars is a context of variables that also contain a list of processors that go with the mapping.
type Vars struct {
	Mapping map[string]interface{}

	ProcessorsKey string
	Processors    []map[string]interface{}
}

// VarsCallback is callback called when the current vars state changes.
type VarsCallback func([]Vars)

// Controller manages the state of the providers current context.
type Controller struct {
	contextProviders map[string]*contextProviderState
	dynamicProviders map[string]*dynamicProviderState
}

// New creates a new controller.
func New(c *config.Config) (*Controller, error) {
	var providersCfg Config
	err := c.Unpack(&providersCfg)
	if err != nil {
		return nil, errors.New(err, "failed to unpack providers config", errors.TypeConfig)
	}

	// build all the context providers
	contextProviders := map[string]*contextProviderState{}
	for name, builder := range Providers.contextProviders {
		pCfg, ok := providersCfg.Providers[name]
		if ok {
			var providerCfg ProviderConfig
			err := pCfg.Unpack(&providerCfg)
			if err != nil {
				return nil, errors.New(err, fmt.Sprintf("failed to unpack provider '%s' config", name), errors.TypeConfig, errors.M("provider", name))
			}
			if providerCfg.Enabled != nil && !*providerCfg.Enabled {
				// explicitly disabled; skipping
				continue
			}
		}
		provider, err := builder(pCfg)
		if err != nil {
			return nil, errors.New(err, fmt.Sprintf("failed to build provider '%s'", name), errors.TypeConfig, errors.M("provider", name))
		}
		contextProviders[name] = &contextProviderState{
			provider: provider,
		}
	}

	// build all the dynamic providers
	dynamicProviders := map[string]*dynamicProviderState{}
	for name, builder := range Providers.dynamicProviders {
		pCfg, ok := providersCfg.Providers[name]
		if ok {
			var providerCfg ProviderConfig
			err := pCfg.Unpack(&providerCfg)
			if err != nil {
				return nil, errors.New(err, fmt.Sprintf("failed to unpack provider '%s' config", name), errors.TypeConfig, errors.M("provider", name))
			}
			if providerCfg.Enabled != nil && !*providerCfg.Enabled {
				// explicitly disabled; skipping
				continue
			}
		}
		provider, err := builder(pCfg)
		if err != nil {
			return nil, errors.New(err, fmt.Sprintf("failed to build provider '%s'", name), errors.TypeConfig, errors.M("provider", name))
		}
		dynamicProviders[name] = &dynamicProviderState{
			provider: provider,
			mappings: map[string]Vars{},
		}
	}

	return &Controller{
		contextProviders: contextProviders,
		dynamicProviders: dynamicProviders,
	}, nil
}

// Run runs the controller.
func (c *Controller) Run(ctx context.Context, cb VarsCallback) error {
	notify := make(chan bool, 5000)
	localCtx, cancel := context.WithCancel(ctx)

	// run all the enabled context providers
	for name, state := range c.contextProviders {
		state.Context = localCtx
		state.signal = notify
		err := state.provider.Run(state)
		if err != nil {
			cancel()
			return errors.New(err, fmt.Sprintf("failed to run provider '%s'", name), errors.TypeConfig, errors.M("provider", name))
		}
	}

	// run all the enabled dynamic providers
	for name, state := range c.dynamicProviders {
		state.Context = localCtx
		state.signal = notify
		err := state.provider.Run(state)
		if err != nil {
			cancel()
			return errors.New(err, fmt.Sprintf("failed to run provider '%s'", name), errors.TypeConfig, errors.M("provider", name))
		}
	}

	go func() {
		for {
			// performs debounce of notifies; accumulates them into 100 millisecond chunks
			changed := false
			for {
				exitloop := false
				select {
				case <-ctx.Done():
					cancel()
					return
				case <-notify:
					changed = true
				case <-time.After(100 * time.Millisecond):
					exitloop = true
					break
				}
				if exitloop {
					break
				}
			}
			if !changed {
				continue
			}

			// build the vars list of mappings
			vars := make([]Vars, 1)
			mapping := map[string]interface{}{}
			for name, state := range c.contextProviders {
				mapping[name] = state.Current()
			}
			vars[0] = Vars{
				Mapping: mapping,
			}

			// add to the vars list for each dynamic providers mappings
			for name, state := range c.dynamicProviders {
				for _, mappings := range state.Mappings() {
					local := copy(mapping)
					local[name] = mappings.Mapping
					vars = append(vars, Vars{
						Mapping:       local,
						ProcessorsKey: name,
						Processors:    mappings.Processors,
					})
				}
			}

			// execute the callback
			cb(vars)
		}
	}()

	return nil
}

type contextProviderState struct {
	context.Context

	provider ContextProvider
	lock     sync.RWMutex
	mapping  map[string]interface{}
	signal   chan bool
}

// Set sets the current mapping.
func (c *contextProviderState) Set(mapping map[string]interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if reflect.DeepEqual(c.mapping, mapping) {
		// same mapping; no need to update and signal
		return
	}
	c.mapping = mapping
	c.signal <- true
}

// Current returns the current mapping.
func (c *contextProviderState) Current() map[string]interface{} {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.mapping
}

type dynamicProviderState struct {
	context.Context

	provider DynamicProvider
	lock     sync.RWMutex
	mappings map[string]Vars
	signal   chan bool
}

// AddOrUpdate adds or updates the current mapping for the dynamic provider.
func (c *dynamicProviderState) AddOrUpdate(id string, mapping map[string]interface{}, processors []map[string]interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()
	curr, ok := c.mappings[id]
	if ok && reflect.DeepEqual(curr.Mapping, mapping) && reflect.DeepEqual(curr.Processors, processors) {
		// same mapping; no need to update and signal
		return
	}
	c.mappings[id] = Vars{
		Mapping:    mapping,
		Processors: processors,
	}
	c.signal <- true
}

// Remove removes the current mapping for the dynamic provider.
func (c *dynamicProviderState) Remove(id string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	_, exists := c.mappings[id]
	if exists {
		// existed; remove and signal
		delete(c.mappings, id)
		c.signal <- true
	}
}

// Mappings returns the current mappings.
func (c *dynamicProviderState) Mappings() []Vars {
	c.lock.RLock()
	defer c.lock.RUnlock()

	mappings := make([]Vars, 0)
	ids := make([]string, 0)
	for name := range c.mappings {
		ids = append(ids, name)
	}
	sort.Strings(ids)
	for _, name := range ids {
		mappings = append(mappings, c.mappings[name])
	}
	return mappings
}

func copy(d map[string]interface{}) map[string]interface{} {
	c := map[string]interface{}{}
	for k, v := range d {
		c[k] = v
	}
	return c
}
