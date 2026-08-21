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
	"sync"
	"time"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/filebeat/input/v2/statemanager"
	"github.com/elastic/beats/v7/libbeat/statestore"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// globalCache is the process-wide singleton store for cursor inputs.
// All InputManager instances that share a (backend, type) pair use one store
// and one background cleaner goroutine. The store is closed only after the
// last manager releases its reference.
var globalCache = statemanager.NewCache[*store](func(s *store) { s.Release() })

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
// are allowed to add a custom per input configuration ID using the `id`
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

	// mu guards store and release. Both are set once on the first successful
	// Create and cleared on Close.
	mu      sync.Mutex
	store   *store
	release func()
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

// setup opens the store via the process-wide cache on the first call, also
// starting a shared background cleaner goroutine. Caller holds cim.mu.
func (cim *InputManager) setup(inputID string) error {
	if cim.store != nil {
		return nil
	}
	if cim.DefaultCleanTimeout <= 0 {
		cim.DefaultCleanTimeout = 30 * time.Minute
	}

	log := cim.Logger.With("input_type", cim.Type)
	key := cim.StateStore.StoreKey() + "::" + cim.Type

	interval := cim.StateStore.CleanupInterval()

	// Only start the background cleaner when a positive cleanup interval is
	// configured. Tests that use in-memory stores with no GCPeriod get a
	// functional store without the extra goroutine.
	var runFn func(context.Context, *store)
	if interval > 0 {
		runFn = func(ctx context.Context, s *store) {
			runCleaner(ctx, log, s, interval)
		}
	}

	s, release, err := globalCache.Acquire(
		key,
		func() (*store, error) {
			return openStore(log, cim.StateStore, cim.Type, inputID, true)
		},
		runFn,
		nil,
	)
	if err != nil {
		return err
	}
	cim.store = s
	cim.release = release
	return nil
}

// Close releases the manager's reference to the shared store. When the last
// manager sharing a (backend, type) key releases, the background cleaner is
// stopped and the store is closed. Call after all inputs managed by this
// manager have stopped.
func (cim *InputManager) Close() {
	cim.mu.Lock()
	release := cim.release
	cim.release = nil
	cim.store = nil
	cim.mu.Unlock()
	if release != nil {
		release()
	}
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

// lock locks a key for exclusive access and returns a resource that can be used to modify
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
