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
	"github.com/elastic/beats/v7/libbeat/features"
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

	// mu protects stores and cleanerGroup. It is not held while a store opens or
	// its cleaner starts, so a slow backend only delays Create for that store key.
	mu sync.Mutex
	// stores holds one entry per store key. Inputs whose IDs select different
	// backends, as Elasticsearch does, must not share a store.
	stores       map[string]*storeEntry
	cleanerGroup unison.Group // saved from Init() for deferred cleaner start
}

// storeEntry is a store that one Create call opens and the others with the same
// store key wait for.
type storeEntry struct {
	ready chan struct{} // closed once store or err is set
	store *store
	err   error
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

// storeKey identifies the cursor store for this input type and ID.
// The key includes the type because each cursor store loads state for one type.
func (cim *InputManager) storeKey(inputID string) string {
	return cim.StateStore.StoreKey(cim.Type, inputID) + "::" + cim.Type
}

// init opens the store for inputID, or returns the one already opened for that
// key. Inputs whose IDs map to different backends get different stores, so an
// Elasticsearch-backed input never reads or writes another input's index.
// Each newly opened store gets its own cleaner once Init supplied a group.
func (cim *InputManager) init(inputID string) (*store, error) {
	key := cim.storeKey(inputID)

	cim.mu.Lock()
	if e, ok := cim.stores[key]; ok {
		cim.mu.Unlock()
		<-e.ready
		return e.store, e.err
	}
	if cim.DefaultCleanTimeout <= 0 {
		cim.DefaultCleanTimeout = 30 * time.Minute
	}
	e := &storeEntry{ready: make(chan struct{})}
	if cim.stores == nil {
		cim.stores = make(map[string]*storeEntry)
	}
	cim.stores[key] = e
	group := cim.cleanerGroup
	cim.mu.Unlock()

	// Open the store and start its cleaner without holding cim.mu: the
	// Elasticsearch store waits for the output configuration before it reads
	// state, and that must not block Create for other input IDs.
	log := cim.Logger.With("input_type", cim.Type)
	e.store, e.err = openStore(log, cim.StateStore, cim.Type, inputID, true)

	// For ES-backed inputs the cleaner is deferred from Init() to here because
	// the store isn't opened until init() is called with the inputID.
	if e.err == nil && group != nil {
		if err := cim.startCleaner(group, e.store); err != nil {
			e.store, e.err = nil, err
		}
	}

	if e.err != nil {
		// Forget the failed entry so that a later Create can try again.
		cim.mu.Lock()
		delete(cim.stores, key)
		cim.mu.Unlock()
	}
	close(e.ready)

	return e.store, e.err
}

// Init starts background processes for deleting old entries from the
// persistent store if mode is ModeRun.
// For ES-backed inputs, store creation is deferred to Create() where the
// inputID is known, so Init() only saves the group for later use.
func (cim *InputManager) Init(group unison.Group) error {
	if features.IsElasticsearchStateStoreEnabledForInput(cim.Type) {
		cim.mu.Lock()
		cim.cleanerGroup = group
		cim.mu.Unlock()
		return nil
	}

	s, err := cim.init("")
	if err != nil {
		return err
	}
	return cim.startCleaner(group, s)
}

// startCleaner launches the background cleaner goroutine that removes stale
// entries from store. It runs one cleaner per store.
func (cim *InputManager) startCleaner(group unison.Group, store *store) error {
	log := cim.Logger.With("input_type", cim.Type)

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
		// Release the reference opened by init() and the one retained above:
		// the cleaner outlives every input using this store.
		defer store.Release()
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
		store.Release()
		return fmt.Errorf("can not start registry cleanup process: %w", err)
	}

	<-waitRunning
	return nil
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

	store, err := cim.init(settings.ID)
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
		store:        store,
		sources:      sources,
		input:        inp,
		cleanTimeout: settings.CleanInactive,
	}, nil
}

// lock gives the caller exclusive access to the cursor state for key.
// The store stays open until its cleaner stops, which happens after every input
// using it has returned.
func lock(ctx v2.Context, store *store, key string) (*resource, error) {
	resource := store.Get(key)
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
