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

package cursor

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/go-concert/unison"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/statestore"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// InputManager is used to create, manage, and coordinate stateful inputs and
// their persistent state.
// The InputManager ensures that only one input can be active for a unique source.
// If two inputs have overlapping sources, both can still collect data, but
// only one input will collect from the common source.
//
// The InputManager automatically cleans up old entries without an active
// input, and without any pending update operations for the persistent store.
//
// The Type field is used to create the key name in the persistent store. Users
// are allowed to add a custome per input configuration ID using the `id`
// setting, to collect the same source multiple times, but with different
// state. The key name in the persistent store becomes <Type>-[<ID>]-<Source Name>
type InputManager struct {
	Logger *logp.Logger

	// StateStore gives the InputManager access to the persistent key value store.
	StateStore statestore.States

	// Type must contain the name of the input type. It is used to create the key name
	// for all sources the inputs collect from.
	Type string

	// DefaultCleanTimeout configures the key/value garbage collection interval.
	// The InputManager will only collect keys for the configured 'Type'
	DefaultCleanTimeout time.Duration

	// Configure returns an array of Sources, and a configured Input instances
	// that will be used to collect events from each source.
	Configure func(cfg *conf.C, log *logp.Logger) ([]Source, Input, error)

	// mu guards the deferred setup below. Inputs are created from several
	// reload paths — static config, central management, autodiscovery — which
	// can run concurrently for one input type.
	mu         sync.Mutex
	initedFull bool
	initErr    error
	store      *store
	// cleanerGroup is the task group Init was given, held until an input is
	// created and cleared once the cleaner is running.
	cleanerGroup unison.Group
	closeOnce    sync.Once
}

// Source describe a source the input can collect data from.
// The `Name` method must return an unique name, that will be used to identify
// the source in the persistent state store.
type Source interface {
	Name() string
}

var (
	errNoSourceConfigured = errors.New("no source has been configured")
	errNoInputRunner      = errors.New("no input runner available")
)

// init initializes the state store with a full init (reading all states).
// Caller holds cim.mu.
func (cim *InputManager) init(inputID string) error {
	if cim.initedFull {
		return nil
	}

	if cim.DefaultCleanTimeout <= 0 {
		cim.DefaultCleanTimeout = 30 * time.Minute
	}

	log := cim.Logger.With("input_type", cim.Type)
	cim.store, cim.initErr = openStore(log, cim.StateStore, cim.Type, inputID, true)
	if cim.initErr != nil {
		return cim.initErr
	}
	cim.initedFull = true

	return nil
}

// Init records the task group the registry cleanup process will run in. Opening
// the store and starting that process are deferred to Create.
//
// v2.Loader.Init calls Init on every registered plugin, but a Beat instantiates
// inputs for only a few of them: eager setup here cost one open registry store
// and one idle cleanup goroutine per input type the Beat never runs. That is
// paid per Beat, and elastic-agent runs one Beat receiver per input stream.
//
// Deferring also means a registry that cannot be opened now fails the inputs
// that need it rather than the whole Beat. Elasticsearch-backed inputs already
// worked this way, since they cannot open the store until Create gives them the
// input ID.
func (cim *InputManager) Init(group unison.Group) error {
	cim.mu.Lock()
	defer cim.mu.Unlock()

	cim.cleanerGroup = group
	return nil
}

// setup opens the store and starts the registry cleanup process, once per
// manager. Caller holds cim.mu.
func (cim *InputManager) setup(inputID string) error {
	if err := cim.init(inputID); err != nil {
		return err
	}

	if cim.cleanerGroup == nil {
		return nil
	}
	group := cim.cleanerGroup
	cim.cleanerGroup = nil
	return cim.startCleaner(group)
}

// startCleaner launches the background cleaner goroutine that removes stale
// entries from the persistent store. Caller holds cim.mu.
func (cim *InputManager) startCleaner(group unison.Group) error {
	log := cim.Logger.With("input_type", cim.Type)

	store := cim.store
	cleaner := &cleaner{log: log}
	store.Retain()
	// TL;DR: If Filebeat shuts down too quickly, the function passed to
	// `group.Go` will never run, therefore this instance of store will
	// never be released, locking Filebeat's shutdown process.
	//
	// To circumvent that, we wait for `group.Go` to start our function.
	// See https://github.com/elastic/beats/issues/45034#issuecomment-3238261126
	waitRunning := make(chan struct{})
	err := group.Go(func(canceler context.Context) error {
		waitRunning <- struct{}{}
		defer store.Release()
		interval := cim.StateStore.CleanupInterval()
		if interval <= 0 {
			interval = 5 * time.Minute
		}
		cleaner.run(canceler, store, interval)
		return nil
	})
	if err != nil {
		store.Release()
		return fmt.Errorf("can not start registry cleanup process: %w", err)
	}

	<-waitRunning
	return nil
}

// Close releases the store acquired during Init or Create. It must be called
// after the task group passed to Init has been stopped and all inputs have finished.
func (cim *InputManager) Close() {
	cim.closeOnce.Do(func() {
		if cim.store != nil {
			cim.store.Release()
		}
	})
}

// Create builds a new v2.Input using the provided Configure function.
// The Input will run a go-routine per source that has been configured.
func (cim *InputManager) Create(config *conf.C) (v2.Input, error) {
	settings := struct {
		ID            string        `config:"id"`
		CleanInactive time.Duration `config:"clean_inactive"`
	}{ID: "", CleanInactive: cim.DefaultCleanTimeout}
	if err := config.Unpack(&settings); err != nil {
		return nil, err
	}

	cim.mu.Lock()
	err := cim.setup(settings.ID)
	cim.mu.Unlock()
	if err != nil {
		return nil, err
	}

	sources, inp, err := cim.Configure(config, cim.Logger)
	if err != nil {
		return nil, err
	}
	if len(sources) == 0 {
		return nil, errNoSourceConfigured
	}
	if inp == nil {
		return nil, errNoInputRunner
	}

	return &managedInput{
		manager:      cim,
		userID:       settings.ID,
		sources:      sources,
		input:        inp,
		cleanTimeout: settings.CleanInactive,
	}, nil
}

// Lock locks a key for exclusive access and returns an resource that can be used to modify
// the cursor state and unlock the key.
func (cim *InputManager) lock(ctx v2.Context, key string) (*resource, error) {
	resource := cim.store.Get(key)
	err := lockResource(ctx.Logger, resource, ctx.Cancelation)
	if err != nil {
		resource.Release()
		return nil, err
	}
	return resource, nil
}

func lockResource(log *logp.Logger, resource *resource, canceler v2.Canceler) error {
	if !resource.lock.TryLock() {
		log.Infof("Resource '%v' currently in use, waiting...", resource.key)
		err := resource.lock.LockContext(canceler)
		if err != nil {
			log.Infof("Input for resource '%v' has been stopped while waiting", resource.key)
			return err
		}
	}
	return nil
}

func releaseResource(resource *resource) {
	resource.lock.Unlock()
	resource.Release()
}
