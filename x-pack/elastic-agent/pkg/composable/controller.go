// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package composable

import (
	"context"
	"encoding/json"
	"fmt"

	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/transpiler"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

// VarsCallback is callback called when the current vars state changes.
type VarsCallback func([]transpiler.Vars)

// Controller manages the state of the providers current context.
type Controller interface {
	// Run runs the controller.
	//
	// Cancelling the context stops the controller.
	Run(ctx context.Context, cb VarsCallback) error
}

// controller manages the state of the providers current context.
type controller struct {
	contextProviders map[string]*contextProviderState
	dynamicProviders map[string]*dynamicProviderState
}

// New creates a new controller.
func New(c *config.Config) (Controller, error) {
	l, err := logger.New("composable")
	if err != nil {
		return nil, err
	}
	l.Info("EXPERIMENTAL - Inputs with variables are currently experimental and should not be used in production")

	var providersCfg Config
	if c != nil {
		err := c.Unpack(&providersCfg)
		if err != nil {
			return nil, errors.New(err, "failed to unpack providers config", errors.TypeConfig)
		}
	}

	// build all the context providers
	contextProviders := map[string]*contextProviderState{}
	for name, builder := range Providers.contextProviders {
		pCfg, ok := providersCfg.Providers[name]
		if ok && !pCfg.Enabled() {
			// explicitly disabled; skipping
			continue
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
		if ok && !pCfg.Enabled() {
			// explicitly disabled; skipping
			continue
		}
		provider, err := builder(pCfg)
		if err != nil {
			return nil, errors.New(err, fmt.Sprintf("failed to build provider '%s'", name), errors.TypeConfig, errors.M("provider", name))
		}
		dynamicProviders[name] = &dynamicProviderState{
			provider: provider,
			mappings: map[string]transpiler.Vars{},
		}
	}

	return &controller{
		contextProviders: contextProviders,
		dynamicProviders: dynamicProviders,
	}, nil
}

// Run runs the controller.
func (c *controller) Run(ctx context.Context, cb VarsCallback) error {
	// large number not to block performing Run on the provided providers
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
			vars := make([]transpiler.Vars, 1)
			mapping := map[string]interface{}{}
			for name, state := range c.contextProviders {
				mapping[name] = state.Current()
			}
			vars[0] = transpiler.Vars{
				Mapping: mapping,
			}

			// add to the vars list for each dynamic providers mappings
			for name, state := range c.dynamicProviders {
				for _, mappings := range state.Mappings() {
					local, _ := cloneMap(mapping) // will not fail; already been successfully cloned once
					local[name] = mappings.Mapping
					vars = append(vars, transpiler.Vars{
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
func (c *contextProviderState) Set(mapping map[string]interface{}) error {
	var err error
	mapping, err = cloneMap(mapping)
	if err != nil {
		return err
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	if reflect.DeepEqual(c.mapping, mapping) {
		// same mapping; no need to update and signal
		return nil
	}
	c.mapping = mapping
	c.signal <- true
	return nil
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
	mappings map[string]transpiler.Vars
	signal   chan bool
}

// AddOrUpdate adds or updates the current mapping for the dynamic provider.
func (c *dynamicProviderState) AddOrUpdate(id string, mapping map[string]interface{}, processors []map[string]interface{}) error {
	var err error
	mapping, err = cloneMap(mapping)
	if err != nil {
		return err
	}
	processors, err = cloneMapArray(processors)
	if err != nil {
		return err
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	curr, ok := c.mappings[id]
	if ok && reflect.DeepEqual(curr.Mapping, mapping) && reflect.DeepEqual(curr.Processors, processors) {
		// same mapping; no need to update and signal
		return nil
	}
	c.mappings[id] = transpiler.Vars{
		Mapping:    mapping,
		Processors: processors,
	}
	c.signal <- true
	return nil
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
func (c *dynamicProviderState) Mappings() []transpiler.Vars {
	c.lock.RLock()
	defer c.lock.RUnlock()

	mappings := make([]transpiler.Vars, 0)
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

func cloneMap(source map[string]interface{}) (map[string]interface{}, error) {
	if source == nil {
		return nil, nil
	}
	bytes, err := json.Marshal(source)
	if err != nil {
		return nil, fmt.Errorf("failed to clone: %s", err)
	}
	var dest map[string]interface{}
	err = json.Unmarshal(bytes, &dest)
	if err != nil {
		return nil, fmt.Errorf("failed to clone: %s", err)
	}
	return dest, nil
}

func cloneMapArray(source []map[string]interface{}) ([]map[string]interface{}, error) {
	if source == nil {
		return nil, nil
	}
	bytes, err := json.Marshal(source)
	if err != nil {
		return nil, fmt.Errorf("failed to clone: %s", err)
	}
	var dest []map[string]interface{}
	err = json.Unmarshal(bytes, &dest)
	if err != nil {
		return nil, fmt.Errorf("failed to clone: %s", err)
	}
	return dest, nil
}
